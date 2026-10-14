import AppKit
import Observation
import SwiftUI

// MARK: - 9 · Explorer

@MainActor
@Observable
final class ExplorerModel {
    let home = FileManager.default.homeDirectoryForCurrentUser.path
    var stack: [String] = [FileManager.default.homeDirectoryForCurrentUser.path]
    var entries: [DiskEntry] = []
    var selection: String?
    var incomplete = false
    var loading = false
    var loaded = false
    /// 9c · While measuring: every child in name order, sizes as they come.
    var pending: [String] = []
    var measured: [String: Int64] = [:]
    var expected = 0
    var notice: String?
    var errorMessage: String?

    private let client: EngineClient
    private var job: Task<Void, Never>?
    /// Last measure of each visited folder, so going back is instant.
    private var cache: [String: [DiskEntry]] = [:]
    private var generation = 0

    init(client: EngineClient = EngineClient()) {
        self.client = client
    }

    var current: String { stack.last ?? "/" }
    var total: Int64 { entries.reduce(0) { $0 + $1.sizeBytes } }

    func load() {
        loaded = true
        loading = true
        let path = current
        job?.cancel()
        generation += 1
        let gen = generation
        pending = ((try? FileManager.default.contentsOfDirectory(atPath: path)) ?? [])
            .sorted()
            .map { (path as NSString).appendingPathComponent($0) }
        measured = [:]
        expected = pending.count
        job = Task {
            do {
                let resp = try await client.analyze(path: path) { event in
                    Task { @MainActor in
                        guard self.generation == gen else { return }
                        self.measured[event.path] = event.bytes
                        if let total = event.total { self.expected = total }
                        if !self.pending.contains(event.path) { self.pending.append(event.path) }
                    }
                }
                guard generation == gen, path == current else { return }
                cache[path] = resp.entries
                entries = resp.entries
                incomplete = resp.entries.contains(where: \.incomplete)
                loading = false
            } catch is CancellationError {
                guard generation == gen else { return }
            } catch {
                guard generation == gen else { return }
                errorMessage = error.localizedDescription
                entries = []
                loading = false
            }
        }
    }

    func open(_ entry: DiskEntry) {
        guard entry.isDir else { return }
        stack.append(entry.path)
        selection = nil
        notice = nil
        entries = []
        load()
    }

    func up() {
        guard stack.count > 1 else { return }
        go(to: stack.count - 2)
    }

    func go(to index: Int) {
        guard index < stack.count - 1 else { return }
        let child = stack[index + 1]
        stack.removeSubrange((index + 1)...)
        selection = child
        notice = nil
        if let cached = cache[current] {
            job?.cancel()
            generation += 1
            entries = cached
            incomplete = cached.contains(where: \.incomplete)
            loading = false
        } else {
            entries = []
            load()
        }
    }

    func delete(_ entry: DiskEntry) {
        generation += 1
        let gen = generation
        loading = true
        Task {
            do {
                let target = EngineTarget.picked(path: entry.path, description: entry.name, sizeBytes: entry.sizeBytes)
                let resp = try await client.clean(targets: [target], dryRun: false) { _ in }
                guard generation == gen else { return }
                if let r = resp.results.first, r.success {
                    notice = String(localized: "\(entry.name) supprimé · \(Fmt.bytes(r.bytesFreed)) libérés")
                    selection = nil
                    LastClean.record(resp.results)
                } else {
                    notice = String(localized: "\(entry.name) laissé en place · \(resp.results.first?.error ?? String(localized: "erreur inconnue"))")
                }
            } catch is CancellationError {
                guard generation == gen else { return }
            } catch {
                guard generation == gen else { return }
                errorMessage = error.localizedDescription
            }
            guard generation == gen else { return }
            // Parent folders still hold the old sizes.
            cache.removeAll()
            load()
        }
    }

    func protect(_ entry: DiskEntry) {
        generation += 1
        let gen = generation
        Task {
            do {
                _ = try await client.ignoreAdd(entry.path)
                guard generation == gen else { return }
                notice = String(localized: "\(entry.name) est maintenant protégé")
            } catch {
                guard generation == gen else { return }
                errorMessage = error.localizedDescription
            }
            guard generation == gen else { return }
            load()
        }
    }

    /// "Macintosh HD › vous › Library": the home folder reads as "vous".
    func crumb(_ index: Int) -> String {
        let path = stack[index]
        if path == home { return String(localized: "vous") }
        if path == "/" { return "Macintosh HD" }
        return (path as NSString).lastPathComponent
    }
}

struct ExplorerView: View {
    @Bindable var model: ExplorerModel
    @State private var pendingDelete: DiskEntry?

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            Breadcrumb(model: model)
                .frame(height: 32)

            Text("Explorer")
                .font(.system(size: 30, weight: .light))
                .tracking(-1)
                .padding(.top, 22)
            let measuring = model.loading && model.entries.isEmpty
            if measuring {
                Text("Mesure des tailles… \(Fmt.count(model.measured.count)) sur \(Fmt.count(model.expected))")
                    .foregroundStyle(Palette.secondary)
                    .padding(.top, 4)
                ThinProgress(fraction: model.expected > 0 ? Double(model.measured.count) / Double(model.expected) : 0)
                    .padding(.top, 16)
            } else {
                Text("Triés par taille · \(model.incomplete ? "≥ " : "")\(Fmt.bytes(model.total))")
                    .foregroundStyle(Palette.muted)
                    .padding(.top, 4)
            }

            if let notice = model.notice {
                Text(notice)
                    .font(.system(size: 12))
                    .foregroundStyle(Palette.muted)
                    .padding(.top, 8)
            }

            if measuring {
                MeasuringList(model: model)
                    .padding(.top, 18)
            } else if model.entries.isEmpty {
                StateCard(ring: .full, title: String(localized: "Dossier vide"), text: String(localized: "Rien à mesurer ici.")) {
                    DoneGlyph()
                } action: {
                    Button("Remonter") { model.up() }
                        .buttonStyle(LinkStyle(color: Palette.accent))
                        .disabled(model.stack.count < 2)
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                let largest = max(model.entries.first?.sizeBytes ?? 1, 1)
                ScrollView {
                    LazyVStack(spacing: 0) {
                        ForEach(model.entries) { entry in
                            EntryRow(
                                entry: entry,
                                fraction: Double(entry.sizeBytes) / Double(largest),
                                selected: model.selection == entry.path,
                                onSelect: { model.selection = entry.path },
                                onOpen: { model.open(entry) },
                                onDelete: { pendingDelete = entry }
                            )
                            .contextMenu {
                                Button("Révéler dans le Finder") { reveal(entry.path) }
                                if !entry.ignored {
                                    Button("Protéger ce chemin") { model.protect(entry) }
                                }
                                Divider()
                                Button("Supprimer…", role: .destructive) { pendingDelete = entry }
                                    .disabled(!entry.removable)
                            }
                        }
                    }
                }
                .padding(.top, 26)
                .opacity(model.loading ? 0.5 : 1)
            }

            HStack {
                Button("‹ Remonter") { model.up() }
                    .buttonStyle(QuietStyle())
                    .disabled(model.stack.count < 2)
                    .keyboardShortcut(.upArrow, modifiers: .command)
                Spacer()
                if measuring {
                    Text("Le tri par taille se fait à la fin de la mesure.")
                        .font(.system(size: 12))
                        .foregroundStyle(Palette.muted)
                }
            }
            .padding(.top, 12)
        }
        .padding(EdgeInsets(top: 32, leading: 72, bottom: 32, trailing: 72))
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .task { if !model.loaded { model.load() } }
        .overlay {
            if let entry = pendingDelete {
                DeleteModal(
                    name: entry.name,
                    path: entry.path,
                    sizeBytes: entry.sizeBytes,
                    incomplete: entry.incomplete,
                    onCancel: { pendingDelete = nil },
                    onDelete: {
                        pendingDelete = nil
                        model.delete(entry)
                    }
                )
            }
        }
        .errorAlert($model.errorMessage)
    }

    private func reveal(_ path: String) {
        NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: path)])
    }
}

/// 9b · Confirmation before deleting one folder from Explorer.
struct DeleteModal: View {
    var name: String
    var path: String
    var sizeBytes: Int64
    var incomplete: Bool
    var onCancel: () -> Void
    var onDelete: () -> Void

    var body: some View {
        ZStack {
            Palette.scrim
                .ignoresSafeArea()
                .onTapGesture(perform: onCancel)

            VStack(alignment: .leading, spacing: 0) {
                Text("Supprimer « \(name) » ?")
                    .font(.system(size: 20, weight: .semibold))
                    .tracking(-0.3)
                    .lineLimit(2)
                Text(displayPath(path))
                    .font(.system(size: 12, design: .monospaced))
                    .foregroundStyle(Palette.muted)
                    .lineLimit(1)
                    .truncationMode(.middle)
                    .padding(.top, 8)
                BigNumber(text: (incomplete ? "≥ " : "") + Fmt.bytes(sizeBytes), size: 46)
                    .padding(.top, 20)
                Text("Suppression définitive.")
                    .foregroundStyle(Palette.muted)
                    .padding(.top, 6)

                HStack(spacing: 12) {
                    Spacer()
                    Button("Annuler", action: onCancel)
                        .buttonStyle(SecondaryPillStyle(height: 42))
                        .keyboardShortcut(.cancelAction)
                    Button("Supprimer", action: onDelete)
                        .buttonStyle(DangerPillStyle())
                }
                .padding(.top, 26)
            }
            .padding(EdgeInsets(top: 30, leading: 32, bottom: 24, trailing: 32))
            .frame(width: 440, alignment: .leading)
            .background(RoundedRectangle(cornerRadius: 24).fill(Palette.surface))
            .shadow(color: Color(hex: 0x2B3752, opacity: 0.18), radius: 35, y: 30)
            .accessibilityElement(children: .contain)
            .accessibilityAddTraits(.isModal)
        }
        .transition(.opacity)
    }
}

/// 9c · Children in name order while they are measured. No actions yet.
struct MeasuringList: View {
    @Bindable var model: ExplorerModel

    var body: some View {
        let largest = max(model.measured.values.max() ?? 1, 1)
        ScrollView {
            LazyVStack(spacing: 0) {
                ForEach(model.pending, id: \.self) { path in
                    HStack(spacing: 16) {
                        Text((path as NSString).lastPathComponent)
                            .font(.system(size: 15))
                            .lineLimit(1)
                            .truncationMode(.middle)
                            .frame(width: 200, alignment: .leading)
                        ZStack(alignment: .leading) {
                            Rectangle().fill(Palette.line).frame(height: 2)
                            if let size = model.measured[path] {
                                GeometryReader { geo in
                                    Capsule().fill(Palette.bar)
                                        .frame(width: max(4, geo.size.width * Double(size) / Double(largest)), height: 4)
                                        .frame(maxHeight: .infinity)
                                }
                            }
                        }
                        .frame(height: 4)
                        Group {
                            if let size = model.measured[path] {
                                Text(Fmt.bytes(size)).font(.system(size: 15)).monospacedDigit()
                            } else {
                                Capsule().fill(Palette.accent.opacity(0.12)).frame(width: 52, height: 10)
                                    .accessibilityLabel("Taille en cours de mesure")
                            }
                        }
                        .frame(width: 88, alignment: .trailing)
                        Spacer().frame(width: 190)
                    }
                    .padding(.horizontal, 12)
                    .frame(height: 54)
                    .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
                }
            }
        }
    }
}

/// 3 pt progress line used by the 9c and 10d loading states.
struct ThinProgress: View {
    var fraction: Double

    var body: some View {
        GeometryReader { geo in
            ZStack(alignment: .leading) {
                Capsule().fill(Palette.accent.opacity(0.12))
                Capsule().fill(Palette.bar).frame(width: geo.size.width * min(max(fraction, 0), 1))
            }
        }
        .frame(height: 3)
        .animation(.easeOut(duration: 0.3), value: fraction)
        .accessibilityElement()
        .accessibilityLabel("Progression")
        .accessibilityValue("\(Int(fraction * 100)) %")
    }
}

struct Breadcrumb: View {
    @Bindable var model: ExplorerModel

    var body: some View {
        HStack(spacing: 6) {
            Text("Macintosh HD")
            ForEach(Array(model.stack.indices), id: \.self) { index in
                Text("›")
                let last = index == model.stack.count - 1
                Button(model.crumb(index)) { model.go(to: index) }
                    .buttonStyle(.plain)
                    .fontWeight(last ? .semibold : .regular)
                    .foregroundStyle(last ? Palette.ink : Palette.muted)
                    .disabled(last)
            }
        }
        .font(.system(size: 13))
        .foregroundStyle(Palette.muted)
        .lineLimit(1)
    }
}

struct EntryRow: View {
    var entry: DiskEntry
    var fraction: Double
    var selected: Bool
    var onSelect: () -> Void
    var onOpen: () -> Void
    var onDelete: () -> Void

    var body: some View {
        HStack(spacing: 16) {
            HStack(spacing: 6) {
                Text(entry.name)
                    .font(.system(size: 15))
                    .lineLimit(1)
                    .truncationMode(.middle)
                if entry.ignored {
                    Image(systemName: "lock.fill")
                        .font(.system(size: 10))
                        .foregroundStyle(Palette.moderate)
                        .accessibilityLabel("protégé")
                }
            }
            .frame(width: 200, alignment: .leading)

            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    Rectangle().fill(Palette.line).frame(height: 2)
                    Capsule().fill(Palette.bar).frame(width: max(4, geo.size.width * fraction), height: 4)
                }
                .frame(maxHeight: .infinity)
            }
            .frame(height: 4)

            Text((entry.incomplete ? "≥ " : "") + Fmt.bytes(entry.sizeBytes))
                .font(.system(size: 15))
                .monospacedDigit()
                .frame(width: 88, alignment: .trailing)
                .help(entry.incomplete ? "Taille minimale : certains sous-dossiers sont illisibles sans accès complet au disque." : "")

            HStack(spacing: 14) {
                // Révéler and Protéger live in the right-click menu (n3).
                if selected {
                    Button("Supprimer", action: onDelete)
                        .buttonStyle(LinkStyle(color: Palette.deleteLabel, size: 12))
                        .fontWeight(.semibold)
                        .disabled(!entry.removable)
                        .help(entry.removable ? "" : entry.ignored ? "Chemin protégé" : "Cet utilisateur ne peut pas le supprimer")
                }
                if entry.isDir {
                    Button(action: onOpen) {
                        Image(systemName: "chevron.right")
                            .font(.system(size: 11, weight: .semibold))
                            .foregroundStyle(Palette.muted)
                            .frame(width: 24, height: 30)
                            .contentShape(Rectangle())
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Ouvrir \(entry.name)")
                }
            }
            .frame(width: 190, alignment: .trailing)
        }
        .padding(.horizontal, 12)
        .frame(height: 54)
        .background(RoundedRectangle(cornerRadius: 10).fill(selected ? Palette.accent.opacity(0.08) : .clear))
        .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
        .contentShape(Rectangle())
        .onTapGesture(count: 2) { if entry.isDir { onOpen() } }
        .onTapGesture(count: 1) { onSelect() }
        .accessibilityElement(children: .contain)
        .accessibilityLabel("\(entry.name), \(Fmt.bytes(entry.sizeBytes))")
        .accessibilityAddTraits(selected ? .isSelected : [])
        .accessibilityAction { onSelect() }
    }
}

// MARK: - 10 · Projets

@MainActor
@Observable
final class ProjectsModel {
    var projects: [DevProject] = []
    var roots: [String] = []
    var selected: Set<String> = []
    var state = LoadState.idle
    var cleaning = false
    var cleanTally = ProgressTally()
    var notice: String?
    var errorMessage: String?
    /// Set once the user cancels, so revisiting the tab does not rescan on its own.
    var cancelled = false
    /// 10d · While searching: projects found so far, progress, folder being read.
    var partial: [DevProject] = []
    var searchDone = 0
    var searchTotal = 0
    var reading: String?

    private let client: EngineClient
    private var job: Task<Void, Never>?
    private var generation = 0
    /// Separate from scan/protect so those actions cannot leave `cleaning` stuck.
    private var cleanGeneration = 0

    init(client: EngineClient = EngineClient()) {
        self.client = client
    }

    var chosenProjects: [DevProject] { projects.filter { selected.contains($0.id) } }
    var chosenArtifacts: [ProjectArtifact] { chosenProjects.flatMap(\.artifacts) }
    var chosenBytes: Int64 { chosenProjects.reduce(0) { $0 + $1.totalBytes } }

    func scan() {
        state = .loading
        notice = nil
        job?.cancel()
        generation += 1
        let gen = generation
        partial = []
        searchDone = 0
        searchTotal = 0
        reading = nil
        job = Task {
            do {
                let resp = try await client.purge { event in
                    Task { @MainActor in
                        guard self.generation == gen else { return }
                        self.apply(event)
                    }
                }
                guard generation == gen else { return }
                projects = resp.projects
                roots = resp.roots
                // Preselect a project only when every artifact in it is old.
                selected = Set(resp.projects.filter { $0.artifacts.allSatisfy(\.selected) }.map(\.id))
                state = .ready
            } catch is CancellationError {
                guard generation == gen else { return }
                state = projects.isEmpty ? .idle : .ready
            } catch {
                guard generation == gen else { return }
                errorMessage = error.localizedDescription
                state = .idle
            }
        }
    }

    /// Folds one engine event into the 10d partial list.
    func apply(_ event: ProgressEvent) {
        switch event.event {
        case "roots":
            roots = (event.detail ?? "").split(separator: "\n").map(String.init)
        case "found":
            let root = (event.path as NSString).deletingLastPathComponent
            let artifact = ProjectArtifact(path: event.path, kind: (event.path as NSString).lastPathComponent,
                                           sizeBytes: event.bytes, safety: "", selected: false)
            if let i = partial.firstIndex(where: { $0.root == root }) {
                let p = partial[i]
                partial[i] = DevProject(root: root, name: p.name, type: p.type,
                                        totalBytes: p.totalBytes + event.bytes, artifacts: p.artifacts + [artifact])
            } else {
                partial.append(DevProject(root: root, name: (root as NSString).lastPathComponent,
                                          type: event.detail ?? "Unknown", totalBytes: event.bytes, artifacts: [artifact]))
            }
            reading = root
        case "progress":
            searchDone = event.items
            searchTotal = event.total ?? searchTotal
        default:
            break
        }
    }

    func cancel() {
        cancelled = true
        generation += 1
        job?.cancel()
        job = nil
        if state == .loading {
            state = projects.isEmpty ? .idle : .ready
        }
    }

    func toggle(_ project: DevProject) {
        if selected.contains(project.id) { selected.remove(project.id) } else { selected.insert(project.id) }
    }

    func protect(_ project: DevProject) {
        generation += 1
        let gen = generation
        Task {
            do {
                _ = try await client.ignoreAdd(project.root)
                guard generation == gen else { return }
                projects.removeAll { $0.id == project.id }
                selected.remove(project.id)
                notice = String(localized: "\(project.name) est maintenant protégé")
            } catch {
                guard generation == gen else { return }
                errorMessage = error.localizedDescription
            }
        }
    }

    func clean() {
        let batch = chosenArtifacts
        guard !batch.isEmpty else { return }
        cleaning = true
        cleanTally = ProgressTally()
        cleanGeneration += 1
        let gen = cleanGeneration
        Task {
            defer {
                if cleanGeneration == gen { cleaning = false }
            }
            do {
                let targets = batch.map {
                    EngineTarget.picked(path: $0.path, description: $0.kind, sizeBytes: $0.sizeBytes, safety: $0.safety)
                }
                let resp = try await client.clean(targets: targets, dryRun: false) { event in
                    Task { @MainActor in
                        guard self.cleanGeneration == gen else { return }
                        self.cleanTally.add(event)
                    }
                }
                guard cleanGeneration == gen else { return }
                let ok = resp.results.filter(\.success)
                let freed = ok.reduce(0) { $0 + $1.bytesFreed }
                let failed = resp.results.count - ok.count
                notice = failed > 0
                    ? String(localized: "\(Fmt.bytes(freed)) libérés · \(failed) dossiers laissés en place")
                    : String(localized: "\(Fmt.bytes(freed)) libérés")
                LastClean.record(resp.results)
                let removed = Set(ok.map(\.path))
                projects = projects.compactMap { p in
                    let rest = p.artifacts.filter { !removed.contains($0.path) }
                    guard !rest.isEmpty else { return nil }
                    return DevProject(root: p.root, name: p.name, type: p.type, totalBytes: rest.reduce(0) { $0 + $1.sizeBytes }, artifacts: rest)
                }
                selected = selected.intersection(Set(projects.map(\.id)))
            } catch {
                guard cleanGeneration == gen else { return }
                errorMessage = error.localizedDescription
            }
        }
    }
}

struct ProjectsView: View {
    @Bindable var model: ProjectsModel
    @Bindable var app: AppModel

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(alignment: .firstTextBaseline) {
                Text("Projets")
                    .font(.system(size: 30, weight: .light))
                    .tracking(-1)
                Spacer()
                if model.state == .loading {
                    Button("Annuler") { model.cancel() }
                        .buttonStyle(LinkStyle())
                        .keyboardShortcut(.cancelAction)
                } else if model.state == .ready {
                    Button("Rescanner") { model.scan() }
                        .buttonStyle(LinkStyle())
                        .disabled(model.cleaning)
                        .padding(.trailing, 18)
                    Text("\(String(localized: "\(model.chosenProjects.count) sélectionnés")) · \(Fmt.bytes(model.chosenBytes))")
                        .font(.system(size: 14))
                        .foregroundStyle(Palette.muted)
                        .monospacedDigit()
                }
            }
            .padding(.top, 36)
            .padding(.bottom, 6)
            if model.state == .loading {
                Text(searchLine)
                    .foregroundStyle(Palette.secondary)
                    .lineLimit(1)
                    .truncationMode(.middle)
                ThinProgress(fraction: model.searchTotal > 0 ? Double(model.searchDone) / Double(model.searchTotal) : 0)
                    .padding(.top, 16)
                    .padding(.bottom, 14)
            } else {
                Text("Le code source reste intact.")
                    .foregroundStyle(Palette.muted)
                    .padding(.bottom, 16)
            }

            switch model.state {
            case .idle:
                ReadyCard(text: String(localized: "Cherche node_modules, target, .venv, Pods…")) { model.scan() }
            case .loading:
                searching
            case .ready where model.projects.isEmpty:
                StateCard(ring: .full, title: String(localized: "Tout est propre"), text: model.notice ?? String(localized: "Aucun dossier de build trouvé.")) {
                    DoneGlyph()
                } action: {
                    Button("Rescanner") { model.scan() }
                        .buttonStyle(LinkStyle(color: Palette.accent))
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            case .ready:
                list
            }
        }
        .padding(.horizontal, 72)
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .overlay {
            if model.cleaning {
                let tally = model.cleanTally
                AnalysingCard(
                    text: tally.items > 0
                        ? String(localized: "Nettoyage… · \(Fmt.count(tally.items)) · \(Fmt.bytes(tally.bytes))")
                        : String(localized: "Nettoyage des artefacts…"),
                    mascot: false
                )
            }
        }
        .task { if model.state == .idle && !model.cancelled { model.scan() } }
        .errorAlert($model.errorMessage)
    }

    /// "Recherche des projets dans ~/Projects et ~/Code… 2 trouvés"
    private var searchLine: String {
        let roots = model.roots.map(displayPath)
        let found = model.partial.count
        switch roots.count {
        case 0: return String(localized: "Recherche des projets… \(found) trouvés")
        case 1: return String(localized: "Recherche des projets dans \(roots[0])… \(found) trouvés")
        case 2: return String(localized: "Recherche des projets dans \(roots[0]) et \(roots[1])… \(found) trouvés")
        default: return String(localized: "Recherche des projets dans \(Fmt.count(roots.count)) dossiers… \(found) trouvés")
        }
    }

    /// 10d · Projects as they are found, skeleton rows below, actions off.
    private var searching: some View {
        VStack(spacing: 0) {
            ScrollView {
                LazyVStack(spacing: 0) {
                    ForEach(model.partial) { project in
                        ProjectRow(project: project, checked: false, interactive: false) {}
                    }
                    ForEach([(120, 210), (96, 180), (140, 230)], id: \.0) { title, subtitle in
                        HStack(spacing: 16) {
                            Circle().strokeBorder(Palette.muted.opacity(0.2), lineWidth: 1.4).frame(width: 18, height: 18)
                            VStack(alignment: .leading, spacing: 8) {
                                Capsule().fill(Palette.accent.opacity(0.12)).frame(width: CGFloat(title), height: 12)
                                Capsule().fill(Palette.accent.opacity(0.12)).frame(width: CGFloat(subtitle), height: 8)
                            }
                            .frame(maxWidth: .infinity, alignment: .leading)
                            Capsule().fill(Palette.accent.opacity(0.12)).frame(width: 54, height: 12)
                        }
                        .frame(height: 60)
                        .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
                        .accessibilityHidden(true)
                    }
                }
            }

            HStack(spacing: 16) {
                if let reading = model.reading {
                    HStack(spacing: 0) {
                        Text("Lecture de ")
                        Text(displayPath(reading)).font(.system(size: 12, design: .monospaced))
                        Text("…")
                    }
                    .font(.system(size: 12))
                    .foregroundStyle(Palette.secondary)
                    .lineLimit(1)
                    .truncationMode(.middle)
                }
                Spacer()
                Button("Prévisualiser") {}.buttonStyle(SecondaryPillStyle()).disabled(true)
                Button("Nettoyer") {}.buttonStyle(SecondaryPillStyle()).disabled(true)
            }
            .padding(.top, 10)
            .padding(.bottom, 32)
        }
    }

    private var list: some View {
        VStack(spacing: 0) {
            ScrollView {
                LazyVStack(spacing: 0) {
                    ForEach(model.projects) { project in
                        ProjectRow(project: project, checked: model.selected.contains(project.id)) {
                            model.toggle(project)
                        }
                        .contextMenu {
                            Button("Révéler dans le Finder") {
                                NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: project.root)])
                            }
                            Button("Protéger ce chemin") { model.protect(project) }
                        }
                    }
                }
            }

            HStack(spacing: 28) {
                VStack(alignment: .leading, spacing: 2) {
                    if let notice = model.notice {
                        Text(notice)
                    }
                }
                .font(.system(size: 12))
                .foregroundStyle(Palette.muted)
                .frame(maxWidth: .infinity, alignment: .leading)

                Button("Prévisualiser") { open(.preview) }
                    .buttonStyle(LinkStyle())
                    .fontWeight(.semibold)
                    .disabled(model.chosenProjects.isEmpty || model.cleaning)
                Button("Nettoyer \(Fmt.count(model.chosenProjects.count))") { open(.confirm) }
                    .buttonStyle(PrimaryPillStyle(height: 48, horizontalPadding: 36))
                    .disabled(model.chosenProjects.isEmpty || model.cleaning)
                    .keyboardShortcut(.defaultAction)
            }
            .frame(height: 84)
            .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
        }
    }
}

extension ProjectsView {
    /// Opens the shared panel for the checked projects: 10b to preview,
    /// 10c to confirm. Either way nothing is deleted before "Nettoyer".
    func open(_ mode: PreviewRequest.Mode) {
        func entries(_ projects: [DevProject]) -> [PreviewEntry] {
            projects.flatMap { p in
                p.artifacts.map {
                    PreviewEntry(
                        target: .picked(path: $0.path, description: $0.kind, sizeBytes: $0.sizeBytes, safety: $0.safety),
                        project: p.name
                    )
                }
            }
        }
        let unselected = model.projects.filter { !model.selected.contains($0.id) }
        let count = model.chosenProjects.count
        app.openPreview(PreviewRequest(
            mode: mode,
            confirmFirst: true,
            question: String(localized: "Nettoyer \(count) projets ?"),
            entries: entries(model.chosenProjects),
            skipped: entries(unselected),
            folders: true,
            countsProjects: true,
            actionTitle: String(localized: "Nettoyer \(Fmt.count(model.chosenProjects.count)) · \(Fmt.bytes(model.chosenBytes))"),
            onClean: { [weak model] in model?.clean() }
        ))
    }
}

struct ProjectRow: View {
    var project: DevProject
    var checked: Bool
    /// False while searching (10d): faint circle, full opacity, no toggle.
    var interactive = true
    var onToggle: () -> Void

    private var typeLabel: String? {
        switch project.type {
        case "Unknown": return nil
        case "Node.js": return "Node"
        case "CocoaPods/iOS": return "iOS"
        default: return project.type
        }
    }

    var body: some View {
        Button(action: onToggle) {
            HStack(spacing: 16) {
                if interactive {
                    CheckDot(state: checked ? .on : .off)
                } else {
                    Circle().strokeBorder(Palette.muted.opacity(0.5), lineWidth: 1.4).frame(width: 18, height: 18)
                }
                VStack(alignment: .leading, spacing: 2) {
                    HStack(spacing: 10) {
                        Text(project.name).font(.system(size: 15))
                        if let typeLabel {
                            Text(typeLabel)
                                .font(.system(size: 11))
                                .padding(.horizontal, 8)
                                .frame(height: 20)
                                .background(Capsule().fill(Palette.projectType(project.type)))
                        }
                    }
                    HStack(spacing: 0) {
                        Text(project.artifacts.map(\.kind).joined(separator: ", ") + " · ")
                        Text(displayPath(project.root)).font(.system(size: 11, design: .monospaced))
                        if interactive && !checked {
                            Text(" · non sélectionné")
                        }
                    }
                    .font(.system(size: 11))
                    .foregroundStyle(Palette.muted)
                    .lineLimit(1)
                    .truncationMode(.middle)
                }
                .frame(maxWidth: .infinity, alignment: .leading)
                Text(Fmt.bytes(project.totalBytes))
                    .font(.system(size: 15))
                    .monospacedDigit()
            }
            .frame(height: 60)
            .contentShape(Rectangle())
            .opacity(checked || !interactive ? 1 : 0.5)
            .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
        }
        .buttonStyle(.plain)
        .allowsHitTesting(interactive)
        .accessibilityLabel("\(project.name), \(Fmt.bytes(project.totalBytes))")
        .accessibilityValue(interactive ? (checked ? "Sélectionné" : "Ignoré") : "Trouvé")
    }
}

struct CheckDot: View {
    enum State { case on, off }
    var state: State
    var size: CGFloat = 18

    var body: some View {
        ZStack {
            switch state {
            case .on:
                Circle().fill(Palette.accent)
                Image(systemName: "checkmark").font(.system(size: size * 0.5, weight: .heavy)).foregroundStyle(.white)
            case .off:
                Circle().strokeBorder(Palette.muted, lineWidth: 1.4)
            }
        }
        .frame(width: size, height: size)
        .accessibilityHidden(true)
    }
}
