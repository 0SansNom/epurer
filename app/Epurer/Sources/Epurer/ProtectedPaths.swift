import AppKit
import Observation
import SwiftUI
import UniformTypeIdentifiers

// MARK: - 11 · Chemins protégés

@MainActor
@Observable
final class ProtectedModel {
    /// Dark bar at the bottom with an "Annuler" that reverts the last change.
    struct Toast: Identifiable {
        let id = UUID()
        let text: String
        let undo: @MainActor () -> Void
    }

    /// Why paths were not added (11b · "Déjà couvert", "Déjà exclu").
    enum Note: Hashable {
        case covered(parent: String, children: [String])
        case system([String])
    }

    var paths: [String] = []
    var loaded = false
    var errorMessage: String?
    var toast: Toast?
    var notes: [Note] = []

    private let client: EngineClient
    private var generation = 0
    private var pending: Task<Void, Never>?

    init(client: EngineClient = EngineClient()) {
        self.client = client
    }

    func load() {
        run({ (try await $0.ignoreList().paths, ()) }, reloadOnError: false) { _ in }
    }

    /// Absolute, tilde-expanded form, the same one the engine stores.
    nonisolated static func normalize(_ path: String) -> String {
        URL(fileURLWithPath: (path as NSString).expandingTildeInPath).standardizedFileURL.path
    }

    nonisolated static func covers(_ parent: String, _ path: String) -> Bool {
        path == parent || path.hasPrefix(parent + "/")
    }

    /// The protected path that already covers `path`, if any.
    func coveringPath(_ path: String) -> String? {
        paths.first { Self.covers($0, path) }
    }

    /// Splits a batch into paths to add and paths already covered, either by
    /// the list or by another path of the same batch. Order is kept.
    nonisolated static func plan(_ raws: [String], protected: [String]) -> (add: [String], covered: [(parent: String, child: String)]) {
        let batch = raws.map(normalize)
        var add: [String] = []
        var covered: [(parent: String, child: String)] = []
        for path in batch {
            let parents = protected + batch.filter { $0 != path && covers($0, path) }
            if let parent = parents.first(where: { covers($0, path) }) {
                if parent != path { covered.append((parent, (path as NSString).lastPathComponent)) }
                else if !protected.contains(path) && !add.contains(path) { add.append(path) }
            } else if !add.contains(path) {
                add.append(path)
            }
        }
        return (add, covered)
    }

    func add(_ raws: [String]) {
        let plan = Self.plan(raws, protected: paths)
        var grouped: [String: [String]] = [:]
        plan.covered.forEach { grouped[$0.parent, default: []].append($0.child) }
        let coveredNotes = grouped.keys.sorted().map { Note.covered(parent: $0, children: grouped[$0]!) }
        notes = coveredNotes
        guard !plan.add.isEmpty else { return }

        run({ client in
            var latest: [String] = []
            var added: [String] = []
            var outOfReach: [String] = []
            for path in plan.add {
                let resp = try await client.ignoreAdd(path)
                latest = resp.paths
                if resp.skipped == "system" { outOfReach.append(path) } else { added.append(path) }
            }
            return (latest, (added, outOfReach))
        }) { [weak self] result in
            guard let self else { return }
            let (added, outOfReach) = result
            if !outOfReach.isEmpty { notes = coveredNotes + [.system(outOfReach)] }
            guard !added.isEmpty else { return }
            let text = added.count == 1
                ? String(localized: "\((added[0] as NSString).lastPathComponent) protégé")
                : String(localized: "\(added.count) éléments protégés")
            show(text) { [weak self] in self?.remove(added, undoable: false) }
        }
    }

    func remove(_ removed: [String], undoable: Bool = true) {
        notes = []
        run({ client in
            var latest: [String] = []
            for path in removed { latest = try await client.ignoreRemove(path).paths }
            return (latest, ())
        }) { [weak self] _ in
            guard undoable else { return }
            let text = removed.count == 1
                ? String(localized: "\((removed[0] as NSString).lastPathComponent) retiré")
                : String(localized: "\(removed.count) éléments retirés")
            self?.show(text) { [weak self] in self?.add(removed) }
        }
    }

    func choose() {
        let panel = NSOpenPanel()
        panel.canChooseDirectories = true
        panel.canChooseFiles = true
        panel.allowsMultipleSelection = true
        panel.prompt = String(localized: "Protéger")
        panel.message = String(localized: "Épurer n'analysera ni ne nettoiera jamais ces chemins.")
        guard panel.runModal() == .OK else { return }
        add(panel.urls.map(\.path))
    }

    /// Waits for every dropped item, then adds them as one batch.
    func drop(_ providers: [NSItemProvider]) -> Bool {
        guard !providers.isEmpty else { return false }
        Task {
            var dropped: [String] = []
            for provider in providers {
                let item = try? await provider.loadItem(forTypeIdentifier: UTType.fileURL.identifier)
                if let data = item as? Data, let url = URL(dataRepresentation: data, relativeTo: nil) {
                    dropped.append(url.path)
                } else if let url = item as? URL {
                    dropped.append(url.path)
                }
            }
            add(dropped)
        }
        return true
    }

    private func show(_ text: String, undo: @escaping @MainActor () -> Void) {
        let toast = Toast(text: text, undo: undo)
        self.toast = toast
        Task { [weak self] in
            try? await Task.sleep(for: .seconds(5))
            if self?.toast?.id == toast.id { self?.toast = nil }
        }
    }

    /// Runs one engine job after the previous one. `op` returns the new list
    /// and a value for `done`.
    private func run<R>(
        _ op: @escaping (EngineClient) async throws -> ([String], R),
        reloadOnError: Bool = true,
        then done: @escaping @MainActor (R) -> Void
    ) {
        generation += 1
        let gen = generation
        // Each engine call rewrites ignore.json; run them one at a time so none overwrites another.
        let previous = pending
        pending = Task {
            await previous?.value
            do {
                let (list, value) = try await op(client)
                guard generation == gen else { return }
                paths = list
                done(value)
            } catch {
                guard generation == gen else { return }
                errorMessage = error.localizedDescription
                // A batch may have failed half-way: show what is really on disk.
                if reloadOnError { load() }
            }
            guard generation == gen else { return }
            loaded = true
        }
    }
}

struct ProtectedPathsView: View {
    @Bindable var model: ProtectedModel
    @State private var dropTargeted = false

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(alignment: .top) {
                VStack(alignment: .leading, spacing: 6) {
                    Text("Chemins protégés")
                        .font(.system(size: 30, weight: .light))
                        .tracking(-1)
                    if !model.paths.isEmpty {
                        Text("Ignorés par tous les scans et nettoyages.")
                            .foregroundStyle(Palette.muted)
                    }
                }
                Spacer()
                if !model.paths.isEmpty {
                    Button("Ajouter un dossier…") { model.choose() }
                        .buttonStyle(PrimaryPillStyle(height: 44, horizontalPadding: 26))
                }
            }

            if model.loaded && model.paths.isEmpty {
                emptyState
            } else {
                list
            }
        }
        .padding(EdgeInsets(top: 34, leading: 72, bottom: 28, trailing: 72))
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
        .onDrop(of: [.fileURL], isTargeted: $dropTargeted) { model.drop($0) }
        .overlay(alignment: .bottom) { toast }
        .animation(.easeOut(duration: 0.2), value: model.toast?.id)
        .task { if !model.loaded { model.load() } }
        .errorAlert($model.errorMessage)
    }

    private var list: some View {
        VStack(alignment: .leading, spacing: 0) {
            // Dropping still works anywhere; the target only shows while dragging (11b).
            if dropTargeted {
                HStack(spacing: 10) {
                    Image(systemName: "folder")
                        .foregroundStyle(Palette.accent)
                    Text("Relâchez pour protéger")
                }
                .frame(maxWidth: .infinity, minHeight: 58)
                .background(RoundedRectangle(cornerRadius: 18).fill(Palette.accent.opacity(0.10)))
                .overlay(dropBorder(cornerRadius: 18))
                .padding(.top, 24)
            }

            ForEach(model.notes, id: \.self) { note in
                NoteView(note: note)
                    .padding(.top, 14)
            }

            ScrollView {
                LazyVStack(spacing: 0) {
                    ForEach(model.paths, id: \.self) { path in
                        ProtectedRow(path: path) { model.remove([path]) }
                    }
                }
            }
            .padding(.top, dropTargeted ? 18 : 26)
        }
    }

    private var emptyState: some View {
        VStack(spacing: 0) {
            VStack(spacing: 0) {
                Mascot(mood: .content, size: 84)
                Text(dropTargeted ? "Relâchez pour protéger" : "Rien n’est protégé")
                    .font(.system(size: 18, weight: .semibold))
                    .padding(.top, 10)
                Text("Glissez un dossier ici.")
                    .foregroundStyle(Palette.muted)
                    .padding(.top, 4)
                Button("Choisir un dossier…") { model.choose() }
                    .buttonStyle(PrimaryPillStyle(height: 44, horizontalPadding: 28))
                    .keyboardShortcut(.defaultAction)
                    .padding(.top, 16)
            }
            .frame(width: 520, height: 270)
            .background(RoundedRectangle(cornerRadius: 28).fill(Palette.accent.opacity(dropTargeted ? 0.10 : 0.06)))
            .overlay(dropBorder(cornerRadius: 28))

            ForEach(model.notes, id: \.self) { note in
                NoteView(note: note)
                    .frame(width: 520)
                    .padding(.top, 14)
            }

        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private func dropBorder(cornerRadius: CGFloat) -> some View {
        RoundedRectangle(cornerRadius: cornerRadius)
            .strokeBorder(
                Palette.accent.opacity(dropTargeted ? 1 : 0.4),
                style: StrokeStyle(lineWidth: dropTargeted ? 2 : 1.5, dash: dropTargeted ? [] : [6, 4])
            )
    }

    @ViewBuilder
    private var toast: some View {
        if let toast = model.toast {
            HStack(spacing: 18) {
                Text(toast.text)
                Button("Annuler") {
                    model.toast = nil
                    toast.undo()
                }
                .buttonStyle(.plain)
                .fontWeight(.semibold)
                .foregroundStyle(Color(hex: 0x9CC4F5))
            }
            .font(.system(size: 13))
            .foregroundStyle(.white)
            .padding(.horizontal, 18)
            .frame(height: 44)
            .background(RoundedRectangle(cornerRadius: 14).fill(Palette.ink))
            .shadow(color: Palette.ink.opacity(0.25), radius: 15, y: 14)
            .padding(.bottom, 24)
            .transition(.move(edge: .bottom).combined(with: .opacity))
        }
    }
}

struct NoteView: View {
    var note: ProtectedModel.Note

    var body: some View {
        Group {
            switch note {
            case .covered(let parent, let children):
                (Text(displayPath(parent)).bold() + Text(" est déjà protégé. ")
                    + Self.boldList(children)
                    + Text(children.count > 1 ? " en font partie." : " en fait partie."))
                    .foregroundStyle(Color(hex: 0x7A5210))
            case .system(let paths):
                (Text("Épurer ne peut pas modifier ") + Self.boldList(paths.map(displayPath))
                    + Text(paths.count > 1 ? " : ces chemins sont hors de portée. Inutile de les protéger."
                                           : " : ce chemin est hors de portée. Inutile de le protéger."))
                    .foregroundStyle(Palette.ink)
            }
        }
        .font(.system(size: 12))
        .lineSpacing(3)
        .padding(.horizontal, 14)
        .padding(.vertical, 12)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(RoundedRectangle(cornerRadius: 14).fill(tint))
    }

    /// "a, b et c" with each name in bold.
    static func boldList(_ names: [String]) -> Text {
        names.enumerated().reduce(Text("")) { text, item in
            let sep = item.offset == 0 ? "" : (item.offset == names.count - 1 ? " et " : ", ")
            return text + Text(sep) + Text(item.element).bold()
        }
    }

    private var tint: Color {
        switch note {
        case .covered: return Palette.moderate.opacity(0.16)
        case .system: return Palette.accent.opacity(0.10)
        }
    }
}

struct ProtectedRow: View {
    var path: String
    var onRemove: () -> Void

    @State private var hovering = false

    private var isVolume: Bool { path.hasPrefix("/Volumes/") }

    var body: some View {
        let exists = FileManager.default.fileExists(atPath: path)

        HStack(spacing: 14) {
            Image(systemName: isVolume ? "externaldrive" : "folder")
                .font(.system(size: 15))
                .foregroundStyle(Palette.accent)
                .frame(width: 38, height: 38)
                .background(RoundedRectangle(cornerRadius: 11).fill(Palette.accent.opacity(exists ? 0.12 : 0.07)))
                .opacity(exists ? 1 : 0.6)
                .accessibilityHidden(true)

            VStack(alignment: .leading, spacing: 2) {
                Text((path as NSString).lastPathComponent).font(.system(size: 15))
                Text(displayPath(path))
                    .font(.system(size: 11, design: .monospaced))
                    .foregroundStyle(Palette.muted)
                    .lineLimit(1)
                    .truncationMode(.middle)
            }
            .opacity(exists ? 1 : 0.7)
            .frame(maxWidth: .infinity, alignment: .leading)

            if !exists {
                Text(isVolume ? "Non monté" : "Introuvable")
                    .font(.system(size: 11))
                    .foregroundStyle(Color(hex: 0x8A5A10))
                    .padding(.horizontal, 9)
                    .frame(height: 20)
                    .background(Capsule().fill(Palette.moderate.opacity(0.2)))
                    .help("La protection reste active au retour du volume.")
            }

            HStack(spacing: 16) {
                if hovering || !exists {
                    if exists {
                        Button("Révéler") { reveal() }
                    }
                    Button("Retirer", action: onRemove)
                } else {
                    Menu {
                        Button("Révéler dans le Finder") { reveal() }
                        Button("Retirer", action: onRemove)
                    } label: {
                        Text("···").font(.system(size: 18)).tracking(2).foregroundStyle(Color(hex: 0x9AA5B8))
                    }
                    .menuStyle(.borderlessButton)
                    .menuIndicator(.hidden)
                    .fixedSize()
                    .accessibilityLabel("Actions pour \((path as NSString).lastPathComponent)")
                }
            }
            .buttonStyle(LinkStyle())
            .frame(width: 120, alignment: .trailing)
        }
        .padding(.horizontal, 14)
        .frame(height: 64)
        .background(RoundedRectangle(cornerRadius: 14).fill(hovering ? Palette.accent.opacity(0.08) : .clear))
        .contentShape(Rectangle())
        .onHover { hovering = $0 }
    }

    private func reveal() {
        NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: path)])
    }
}
