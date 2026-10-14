import AppKit
import Foundation
import Observation

enum Destination: Int, CaseIterable, Identifiable {
    case home, safeClean, explorer, projects, protectedPaths

    var id: Int { rawValue }

    var title: String {
        switch self {
        case .home: return String(localized: "Accueil")
        case .safeClean: return String(localized: "Nettoyage sûr")
        case .explorer: return String(localized: "Explorer")
        case .projects: return String(localized: "Projets")
        case .protectedPaths: return String(localized: "Chemins protégés")
        }
    }

    static let main: [Destination] = [.home, .safeClean, .explorer, .projects]
    static let manage: [Destination] = [.protectedPaths]
}

/// Linear flow: Scan → Estimation → Review → Confirm → Nettoyage → Résultat.
enum Stage: Equatable {
    case idle, scanning, report, review, confirm, cleaning, result
}

struct DiskUsage {
    var total: Int64
    var available: Int64

    var used: Int64 { max(total - available, 0) }
    var fraction: Double { total > 0 ? Double(used) / Double(total) : 0 }

    static func current() -> DiskUsage? {
        let url = FileManager.default.homeDirectoryForCurrentUser
        let keys: Set<URLResourceKey> = [.volumeTotalCapacityKey, .volumeAvailableCapacityForImportantUsageKey]
        guard let values = try? url.resourceValues(forKeys: keys),
              let total = values.volumeTotalCapacity,
              let available = values.volumeAvailableCapacityForImportantUsage
        else { return nil }
        return DiskUsage(total: Int64(total), available: available)
    }
}

/// Folds the engine's per-walk progress (items/bytes restart for every root
/// or cleaner) into running totals for the whole job.
struct ProgressTally {
    private(set) var path = ""
    private(set) var items = 0
    private(set) var bytes: Int64 = 0
    private var baseItems = 0
    private var baseBytes: Int64 = 0
    private var lastItems = 0
    private var lastBytes: Int64 = 0

    mutating func add(_ e: ProgressEvent) {
        if e.items <= lastItems {
            baseItems += lastItems
            baseBytes += lastBytes
        }
        lastItems = e.items
        lastBytes = e.bytes
        path = e.path
        items = baseItems + e.items
        bytes = baseBytes + e.bytes
    }
}

/// Where a clean was started from, so "Terminé" / errors return there.
enum CleanOrigin {
    case review, smart
}

enum LoadState {
    case idle, loading, ready
}

/// One line of the "Prévisualiser" panel. `project` is set for build
/// folders, which read as "project / kind".
struct PreviewEntry: Identifiable {
    var id: String { target.path }
    let target: EngineTarget
    var project: String?
}

/// What a "Prévisualiser" button asks for. Shared by Sélection,
/// Confirmation, Nettoyage sûr and Projets. The same panel doubles as the
/// final confirmation for Projets (10c).
struct PreviewRequest {
    enum Mode { case preview, confirm }

    let id = UUID()
    var mode = Mode.preview
    /// When true, the preview's main button leads to the confirm mode
    /// instead of cleaning straight away.
    var confirmFirst = false
    /// Confirm mode heading, e.g. "Nettoyer 5 projets ?".
    var question: String?
    /// What the next "Nettoyer" would delete.
    var entries: [PreviewEntry]
    /// Listed on the screen but not selected: left in place.
    var skipped: [PreviewEntry] = []
    /// Already known to be out of reach for this user.
    var locked: [EngineTarget] = []
    /// Counts folders ("3 dossiers") instead of items.
    var folders = false
    var countsProjects = false
    var actionTitle: String
    var onClean: @MainActor () -> Void
}

struct PreviewState {
    var request: PreviewRequest
    /// nil while the engine's dry run is in progress.
    var results: [CleanResult]?
}

/// The last real clean, kept across launches for the home screen.
enum LastClean {
    static let bytesKey = "lastCleanBytes"
    static let dateKey = "lastCleanDate"

    /// What was actually freed, or nil when nothing was removed.
    static func freed(_ results: [CleanResult]) -> Int64? {
        let ok = results.filter(\.success)
        return ok.isEmpty ? nil : ok.reduce(0) { $0 + $1.bytesFreed }
    }

    static func record(_ results: [CleanResult]) {
        guard let bytes = freed(results) else { return }
        UserDefaults.standard.set(Int(bytes), forKey: bytesKey)
        UserDefaults.standard.set(Date().timeIntervalSince1970, forKey: dateKey)
    }
}

struct DomainTotal: Identifiable {
    var id: String { domain }
    let domain: String
    let bytes: Int64
}

@MainActor
@Observable
final class AppModel {
    var destination: Destination = .home
    var stage: Stage = .idle
    var level: ScanLevel = .standard

    var info: EngineInfo?
    var disk: DiskUsage? = DiskUsage.current()
    var diskAccess = "unknown"

    var targets: [EngineTarget] = []
    var locked: [EngineTarget] = []
    var selected: Set<EngineTarget.ID> = []

    var scanTally = ProgressTally()
    var cleanTally = ProgressTally()

    var results: [CleanResult] = []
    var cleanedTargets: [EngineTarget] = []
    var cleanOrigin = CleanOrigin.review

    /// "Nettoyage sûr": Safe items from a conservative scan.
    var safeState = LoadState.idle
    var safeTargets: [EngineTarget] = []
    var safeTally = ProgressTally()
    /// Set once the user cancels, so revisiting the tab does not rescan on its own.
    var safeCancelled = false

    /// Stacks shown at the bottom of the home screen.
    var detectState = LoadState.idle
    var domains: [DomainStatus] = []

    /// Full-screen "Accès complet au disque" explainer.
    var showPermission = false

    /// Full-screen "Prévisualiser" panel.
    var preview: PreviewState?

    var errorMessage: String?
    var notice: String?
    /// The engine binary is not in the app: only on a broken local build (1c).
    var engineMissing = false

    /// Shared by Accueil, Nettoyage sûr, Explorer, Projets and Chemins protégés.
    let engine = EngineClient()
    private var client: EngineClient { engine }
    private var job: Task<Void, Never>?
    private var detectJob: Task<Void, Never>?
    private var safeJob: Task<Void, Never>?
    /// Separate from `generation` so the safe scan never invalidates a
    /// scan or clean running in the main flow, and vice versa.
    private var safeGeneration = 0
    /// Bumped for every new job so a cancelled one cannot overwrite the next one's state.
    private var generation = 0

    // MARK: Derived

    var accessLimited: Bool { diskAccess != "granted" }

    var totalBytes: Int64 { targets.reduce(0) { $0 + $1.sizeBytes } }
    var lockedBytes: Int64 { locked.reduce(0) { $0 + $1.sizeBytes } }

    var chosen: [EngineTarget] { targets.filter { selected.contains($0.id) } }
    var chosenBytes: Int64 { chosen.reduce(0) { $0 + $1.sizeBytes } }

    var domainTotals: [DomainTotal] {
        Dictionary(grouping: targets, by: \.domain)
            .map { DomainTotal(domain: $0.key, bytes: $0.value.reduce(0) { $0 + $1.sizeBytes }) }
            .sorted { $0.bytes > $1.bytes }
    }

    var cleanFraction: Double {
        let total = cleanedTargets.reduce(0) { $0 + $1.sizeBytes }
        guard total > 0 else { return 0 }
        return min(Double(cleanTally.bytes) / Double(total), 1)
    }

    var freedBytes: Int64 { results.filter(\.success).reduce(0) { $0 + $1.bytesFreed } }

    var safeBytes: Int64 { safeTargets.reduce(0) { $0 + $1.sizeBytes } }

    /// Safe targets grouped by the cleaner that found them, largest first.
    var safeGroups: [(name: String, bytes: Int64)] {
        Dictionary(grouping: safeTargets, by: \.cleaner)
            .map { (name: $0.key.engineLocalized, bytes: $0.value.reduce(0) { $0 + $1.sizeBytes }) }
            .sorted { $0.bytes > $1.bytes }
    }

    /// The level the "Rien trouvé" state offers next, if any.
    var nextLevel: ScanLevel? {
        switch level {
        case .conservative: return .standard
        case .standard: return .aggressive
        case .aggressive: return nil
        }
    }

    func target(at path: String) -> EngineTarget? {
        cleanedTargets.first { $0.path == path }
    }

    // MARK: Actions

    func start() async {
        do {
            let info = try await client.info()
            self.info = info
            diskAccess = info.diskAccess
            engineMissing = false
            let key = "permissionPromptShown"
            if accessLimited && !UserDefaults.standard.bool(forKey: key) {
                UserDefaults.standard.set(true, forKey: key)
                showPermission = true
            }
            detect()
        } catch EngineClientError.binaryMissing {
            engineMissing = true
        } catch {
            errorMessage = error.localizedDescription
        }
    }

    /// Detected stacks for the home footer: System is always scanned and
    /// never listed; at most `limit` are shown, the rest count as "+N".
    func homeStacks(limit: Int = 5) -> (shown: [DomainStatus], extra: Int) {
        let stacks = domains.filter { $0.detected && $0.domain != "System" }
        return (Array(stacks.prefix(limit)), max(stacks.count - limit, 0))
    }

    /// Re-reads Full Disk Access, e.g. when the user comes back from System Settings.
    func refreshAccess() async {
        guard let info = try? await client.info() else { return }
        self.info = info
        diskAccess = info.diskAccess
    }

    func detect() {
        detectState = .loading
        detectJob = Task {
            do {
                domains = try await client.detect().domains
            } catch {
                errorMessage = error.localizedDescription
            }
            detectState = .ready
        }
    }

    func scanSafe() {
        safeState = .loading
        safeCancelled = false
        safeTally = ProgressTally()
        safeJob?.cancel()
        safeGeneration += 1
        let gen = safeGeneration
        safeJob = Task {
            do {
                let response = try await client.scan(level: .conservative) { event in
                    Task { @MainActor in
                        guard self.safeGeneration == gen else { return }
                        self.safeTally.add(event)
                    }
                }
                guard safeGeneration == gen else { return }
                safeTargets = response.targets
                    .filter { $0.safetyLevel == .safe }
                    .sorted { $0.sizeBytes > $1.sizeBytes }
                diskAccess = response.diskAccess
                safeState = .ready
            } catch {
                guard safeGeneration == gen else { return }
                if !(error is CancellationError) {
                    errorMessage = error.localizedDescription
                }
                safeState = .idle
            }
        }
    }

    func cancelSafe() {
        safeCancelled = true
        safeJob?.cancel()
        safeJob = nil
        safeGeneration += 1
        safeState = .idle
    }

    func cleanSafe() {
        clean(batch: safeTargets, origin: .smart)
    }

    /// True when the batch would empty the Trash, which the design calls out.
    func includesTrash(_ targets: [EngineTarget]) -> Bool {
        targets.contains { $0.cleaner == "Trash" }
    }

    // MARK: Prévisualiser

    func openPreview(_ request: PreviewRequest) {
        preview = PreviewState(request: request, results: nil)
        Task {
            do {
                let resp = try await client.clean(targets: request.entries.map(\.target), dryRun: true)
                guard preview?.request.id == request.id else { return }
                preview?.results = resp.results
            } catch {
                guard preview?.request.id == request.id else { return }
                preview = nil
                errorMessage = error.localizedDescription
            }
        }
    }

    func closePreview() {
        preview = nil
    }

    func setPreviewMode(_ mode: PreviewRequest.Mode) {
        preview?.request.mode = mode
    }

    func previewSelection(then onClean: @escaping @MainActor () -> Void) {
        let skipped = targets.filter { !selected.contains($0.id) }
        openPreview(PreviewRequest(
            entries: chosen.map { PreviewEntry(target: $0) },
            skipped: skipped.map { PreviewEntry(target: $0) },
            locked: locked,
            actionTitle: String(localized: "Nettoyer \(Fmt.count(chosen.count)) · \(Fmt.bytes(chosenBytes))"),
            onClean: onClean
        ))
    }

    func previewSafe() {
        openPreview(PreviewRequest(
            confirmFirst: true,
            question: String(localized: "Nettoyer \(safeTargets.count) éléments sûrs ?"),
            entries: safeTargets.map { PreviewEntry(target: $0) },
            actionTitle: String(localized: "Nettoyer en un geste · \(Fmt.bytes(safeBytes))"),
            onClean: { [weak self] in self?.cleanSafe() }
        ))
    }

    func scan(level: ScanLevel? = nil) {
        if let level { self.level = level }
        destination = .home
        showPermission = false
        preview = nil
        stage = .scanning
        scanTally = ProgressTally()
        notice = nil
        let level = self.level

        safeJob?.cancel()
        safeJob = nil
        safeGeneration += 1
        job?.cancel()
        generation += 1
        let gen = generation
        let detecting = detectJob
        job = Task {
            // Product rule: a scan started while stacks are being detected waits for them.
            await detecting?.value
            do {
                try Task.checkCancellation()
                let response = try await client.scan(level: level) { event in
                    Task { @MainActor in
                        guard self.generation == gen else { return }
                        self.scanTally.add(event)
                    }
                }
                guard generation == gen else { return }
                targets = response.targets.sorted { $0.sizeBytes > $1.sizeBytes }
                locked = response.locked
                diskAccess = response.diskAccess
                selected = Set(targets.filter { $0.safetyLevel != .dangerous }.map(\.id))
                stage = .report
            } catch {
                guard generation == gen else { return }
                if !(error is CancellationError) {
                    errorMessage = error.localizedDescription
                }
                stage = .idle
            }
        }
    }

    func cancel() {
        job?.cancel()
        job = nil
        generation += 1
    }

    func toggle(_ target: EngineTarget) {
        if selected.contains(target.id) {
            selected.remove(target.id)
        } else {
            selected.insert(target.id)
        }
    }

    func selectAll(_ on: Bool) {
        if on {
            selected = Set(targets.filter { $0.safetyLevel != .dangerous }.map(\.id))
        } else {
            selected = []
        }
    }

    /// `keeping`: results of an earlier pass shown again with this one ("Réessayer l’échec").
    func clean(batch: [EngineTarget]? = nil, origin: CleanOrigin = .review, keeping: [CleanResult] = []) {
        let batch = batch ?? chosen
        guard !batch.isEmpty else { return }
        let earlier = cleanedTargets.filter { t in keeping.contains { $0.path == t.path } }
        preview = nil
        cleanedTargets = batch
        cleanOrigin = origin
        cleanTally = ProgressTally()
        destination = .home
        stage = .cleaning

        // Stop any in-flight safe/home scan so it cannot overwrite clean results.
        safeJob?.cancel()
        safeJob = nil
        safeGeneration += 1
        job?.cancel()
        generation += 1
        let gen = generation
        job = Task {
            do {
                let response = try await client.clean(targets: batch, dryRun: false) { event in
                    Task { @MainActor in
                        guard self.generation == gen else { return }
                        self.cleanTally.add(event)
                    }
                }
                guard generation == gen else { return }
                results = keeping + response.results
                cleanedTargets = earlier + batch
                LastClean.record(results)
                stage = .result
            } catch is CancellationError {
                guard generation == gen else { return }
                notice = String(localized: "Nettoyage interrompu. La sélection précédente est toujours là.")
                returnToOrigin()
            } catch {
                guard generation == gen else { return }
                errorMessage = error.localizedDescription
                returnToOrigin()
            }
            disk = DiskUsage.current()
        }
    }

    /// Runs the failed items of the last clean again, keeping the other results on screen.
    func retryFailed() {
        let failed = Set(results.filter { $0.status == "failed" }.map(\.path))
        clean(batch: cleanedTargets.filter { failed.contains($0.path) },
              origin: cleanOrigin,
              keeping: results.filter { !failed.contains($0.path) })
    }

    func finish() {
        results = []
        cleanedTargets = []
        disk = DiskUsage.current()
        targets = []
        locked = []
        selected = []
        safeTargets = []
        safeState = .idle
        stage = .idle
    }

    private func returnToOrigin() {
        switch cleanOrigin {
        case .review:
            stage = .review
        case .smart:
            stage = .idle
            destination = .safeClean
        }
    }

    func openFullDiskAccessSettings() {
        if let url = URL(string: "x-apple.systempreferences:com.apple.preference.security?Privacy_AllFiles") {
            NSWorkspace.shared.open(url)
        }
    }
}
