import Foundation

enum EngineClientError: LocalizedError {
    case binaryMissing(String)
    case nonZeroExit(Int32, String)
    case decode(String)

    var errorDescription: String? {
        switch self {
        case .binaryMissing(let path):
            return String(localized: "Moteur introuvable. Relancez « make app-bundle » (attendu : \(path)).")
        case .nonZeroExit(let code, let stderr):
            let detail = stderr.isEmpty ? "code \(code)" : stderr
            if detail.localizedCaseInsensitiveContains("permission")
                || detail.localizedCaseInsensitiveContains("operation not permitted") {
                return String(localized: "Accès refusé. Accordez l’accès complet au disque à Épurer dans Réglages, puis réessayez.")
            }
            return String(localized: "Le moteur s’est arrêté (\(code)) : \(detail)")
        case .decode(let detail):
            return String(localized: "Réponse du moteur illisible : \(detail)")
        }
    }
}

/// Clean levels understood by `epurer engine scan --level`.
enum ScanLevel: String, CaseIterable, Identifiable, Sendable {
    case conservative, standard, aggressive

    var id: String { rawValue }

    var label: String {
        switch self {
        case .conservative: return String(localized: "Prudent")
        case .standard: return String(localized: "Standard")
        case .aggressive: return String(localized: "Agressif")
        }
    }

    /// What the level includes, e.g. "Standard : sûrs et modérés".
    var scope: String {
        switch self {
        case .conservative: return String(localized: "Prudent : sûrs uniquement")
        case .standard: return String(localized: "Standard : sûrs et modérés")
        case .aggressive: return String(localized: "Agressif : risqués compris")
        }
    }
}

/// Talks to the Go `epurer engine` subprocess. stdout = final JSON, stderr = NDJSON progress.
/// Cancelling the calling Task terminates the subprocess.
struct EngineClient: Sendable {
    var binaryURL: URL

    static func resolveBinary() -> URL {
        // Prefer Resources/epurer-engine inside the .app (case-insensitive APFS-safe).
        if let resource = Bundle.main.url(forResource: "epurer-engine", withExtension: nil) {
            return resource
        }
        if let bundled = Bundle.main.url(forAuxiliaryExecutable: "epurer-engine") {
            return bundled
        }
        #if DEBUG
        let env = ProcessInfo.processInfo.environment["EPURER_ENGINE"]
        if let env, !env.isEmpty {
            return URL(fileURLWithPath: env)
        }
        #endif
        // Local package layout: app/Epurer/.engine/epurer
        let packageEngine = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent() // Sources/Epurer
            .deletingLastPathComponent() // Sources
            .deletingLastPathComponent() // Epurer package root
            .appendingPathComponent(".engine/epurer")
        if FileManager.default.isExecutableFile(atPath: packageEngine.path) {
            return packageEngine
        }
        return URL(fileURLWithPath: "/usr/local/bin/epurer")
    }

    init(binaryURL: URL = EngineClient.resolveBinary()) {
        self.binaryURL = binaryURL
    }

    func info() async throws -> EngineInfo {
        try await run(arguments: ["engine", "info"])
    }

    func scan(level: ScanLevel, progress: (@Sendable (ProgressEvent) -> Void)? = nil) async throws -> ScanResponse {
        var args = ["engine", "scan", "--level", level.rawValue]
        if progress != nil {
            args.append("--progress")
        }
        return try await run(arguments: args, progress: progress)
    }

    func clean(targets: [EngineTarget], dryRun: Bool, progress: (@Sendable (ProgressEvent) -> Void)? = nil) async throws -> CleanResponse {
        var args = ["engine", "clean"]
        if progress != nil {
            args.append("--progress")
        }
        let body = CleanRequest(protocolVersion: 1, dryRun: dryRun, targets: targets)
        let data = try JSONEncoder().encode(body)
        return try await run(arguments: args, stdin: data, progress: progress)
    }

    func analyze(path: String, progress: (@Sendable (ProgressEvent) -> Void)? = nil) async throws -> AnalyzeResponse {
        try await run(arguments: ["engine", "analyze", path] + (progress == nil ? [] : ["--progress"]), progress: progress)
    }

    func purge(path: String? = nil, progress: (@Sendable (ProgressEvent) -> Void)? = nil) async throws -> PurgeResponse {
        try await run(arguments: ["engine", "purge"] + (path.map { [$0] } ?? []) + (progress == nil ? [] : ["--progress"]), progress: progress)
    }

    func detect() async throws -> DetectResponse {
        try await run(arguments: ["engine", "detect"])
    }

    func ignoreList() async throws -> IgnoreResponse {
        try await run(arguments: ["engine", "ignore", "list"])
    }

    func ignoreAdd(_ path: String) async throws -> IgnoreResponse {
        try await run(arguments: ["engine", "ignore", "add", path])
    }

    func ignoreRemove(_ path: String) async throws -> IgnoreResponse {
        try await run(arguments: ["engine", "ignore", "remove", path])
    }

    private func run<T: Decodable & Sendable>(
        arguments: [String],
        stdin: Data? = nil,
        progress: (@Sendable (ProgressEvent) -> Void)? = nil
    ) async throws -> T {
        let path = binaryURL.path
        guard FileManager.default.isExecutableFile(atPath: path) else {
            throw EngineClientError.binaryMissing(path)
        }

        let job = EngineJob(binaryURL: binaryURL, arguments: arguments, stdin: stdin, progress: progress)
        let output = try await withTaskCancellationHandler {
            try await withCheckedThrowingContinuation { (cont: CheckedContinuation<EngineJob.Output, Error>) in
                DispatchQueue.global(qos: .userInitiated).async {
                    cont.resume(with: Result { try job.runToCompletion() })
                }
            }
        } onCancel: {
            job.cancel()
        }

        // Cancel during shutdown must not discard a finished success: the
        // process may already have exited 0 with a full JSON body.
        if output.status == 0 {
            do {
                return try JSONDecoder().decode(T.self, from: output.stdout)
            } catch {
                if output.cancelled {
                    throw CancellationError()
                }
                throw EngineClientError.decode(error.localizedDescription)
            }
        }
        if output.cancelled {
            throw CancellationError()
        }
        throw EngineClientError.nonZeroExit(output.status, output.stderrText)
    }
}

/// One engine subprocess. Drains stdout and stderr concurrently so a large
/// scan document can never fill the pipe and deadlock the child.
private final class EngineJob: @unchecked Sendable {
    struct Output {
        var status: Int32
        var stdout: Data
        var stderrText: String
        var cancelled: Bool
    }

    private let process = Process()
    private let stdin: Data?
    private let progress: (@Sendable (ProgressEvent) -> Void)?
    private let lock = NSLock()
    private var cancelled = false
    private var started = false

    init(binaryURL: URL, arguments: [String], stdin: Data?, progress: (@Sendable (ProgressEvent) -> Void)?) {
        process.executableURL = binaryURL
        process.arguments = arguments
        var env = ProcessInfo.processInfo.environment
        env["PATH"] = EngineJob.toolPath(current: env["PATH"])
        process.environment = env
        self.stdin = stdin
        self.progress = progress
    }

    /// Apps launched from Finder get PATH=/usr/bin:/bin:/usr/sbin:/sbin, so the
    /// engine would not find node, cargo, brew or docker and would skip those
    /// domains. Add the usual per-user and Homebrew tool directories.
    static func toolPath(current: String?) -> String {
        let home = FileManager.default.homeDirectoryForCurrentUser.path
        var dirs = [
            "/opt/homebrew/bin", "/opt/homebrew/sbin", "/usr/local/bin",
            "\(home)/.cargo/bin", "\(home)/go/bin", "\(home)/.local/bin",
            "\(home)/.bun/bin", "\(home)/.deno/bin", "\(home)/.volta/bin",
            "\(home)/Library/pnpm", "\(home)/.asdf/shims", "\(home)/.local/share/mise/shims",
            "/Applications/Docker.app/Contents/Resources/bin",
        ]
        // Node version managers keep each version in its own directory.
        let fm = FileManager.default
        for fnm in ["\(home)/.local/share/fnm", "\(home)/Library/Application Support/fnm"] {
            dirs.append("\(fnm)/aliases/default/bin")
            if let versions = try? fm.contentsOfDirectory(atPath: "\(fnm)/node-versions") {
                dirs += versions.sorted().reversed().map { "\(fnm)/node-versions/\($0)/installation/bin" }
            }
        }
        let nvm = "\(home)/.nvm/versions/node"
        if let versions = try? fm.contentsOfDirectory(atPath: nvm) {
            dirs += versions.sorted().reversed().map { "\(nvm)/\($0)/bin" }
        }
        dirs += (current ?? "").split(separator: ":").map(String.init)
        dirs += ["/usr/bin", "/bin", "/usr/sbin", "/sbin"]

        var seen = Set<String>()
        return dirs.filter { !$0.isEmpty && seen.insert($0).inserted }.joined(separator: ":")
    }

    func cancel() {
        lock.lock()
        cancelled = true
        let running = started
        lock.unlock()
        if running, process.isRunning {
            process.terminate()
        }
    }

    func runToCompletion() throws -> Output {
        let outPipe = Pipe()
        let errPipe = Pipe()
        let inPipe = Pipe()
        process.standardOutput = outPipe
        process.standardError = errPipe
        process.standardInput = stdin == nil ? FileHandle.nullDevice : inPipe

        lock.lock()
        if cancelled {
            lock.unlock()
            return Output(status: -1, stdout: Data(), stderrText: "", cancelled: true)
        }
        do {
            try process.run()
        } catch {
            lock.unlock()
            throw error
        }
        started = true
        lock.unlock()

        let group = DispatchGroup()
        var outData = Data()
        var errText = ""

        group.enter()
        DispatchQueue.global(qos: .userInitiated).async {
            outData = outPipe.fileHandleForReading.readDataToEndOfFile()
            group.leave()
        }

        group.enter()
        DispatchQueue.global(qos: .userInitiated).async { [progress] in
            let handle = errPipe.fileHandleForReading
            var buffer = Data()
            var other = Data()
            while true {
                let chunk = handle.availableData
                if chunk.isEmpty { break }
                buffer.append(chunk)
                while let nl = buffer.firstIndex(of: 0x0A) {
                    let line = buffer.subdata(in: buffer.startIndex..<nl)
                    buffer.removeSubrange(buffer.startIndex...nl)
                    if let event = try? JSONDecoder().decode(ProgressEvent.self, from: line) {
                        progress?(event)
                    } else {
                        other.append(line)
                        other.append(0x0A)
                    }
                }
            }
            other.append(buffer)
            errText = String(data: other, encoding: .utf8) ?? ""
            group.leave()
        }

        if let stdin {
            inPipe.fileHandleForWriting.write(stdin)
            try? inPipe.fileHandleForWriting.close()
        }

        process.waitUntilExit()
        group.wait()

        lock.lock()
        let wasCancelled = cancelled
        lock.unlock()

        return Output(
            status: process.terminationStatus,
            stdout: outData,
            stderrText: errText.trimmingCharacters(in: .whitespacesAndNewlines),
            cancelled: wasCancelled
        )
    }
}
