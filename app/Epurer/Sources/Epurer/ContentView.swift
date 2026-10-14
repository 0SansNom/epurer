import AppKit
import SwiftUI

struct ContentView: View {
    @State private var model: AppModel
    @State private var explorer: ExplorerModel
    @State private var projects: ProjectsModel
    @State private var protected: ProtectedModel

    init() {
        let app = AppModel()
        _model = State(initialValue: app)
        _explorer = State(initialValue: ExplorerModel(client: app.engine))
        _projects = State(initialValue: ProjectsModel(client: app.engine))
        _protected = State(initialValue: ProtectedModel(client: app.engine))
    }

    var body: some View {
        HStack(spacing: 0) {
            Sidebar(model: model)
            detail
                .frame(maxWidth: .infinity, maxHeight: .infinity)
        }
        .foregroundStyle(Palette.ink)
        .font(.system(size: 13))
        .background(AppBackground(scenery: scenery))
        .ignoresSafeArea()
        .task { await model.start() }
        .onReceive(NotificationCenter.default.publisher(for: NSApplication.didBecomeActiveNotification)) { _ in
            Task { await model.refreshAccess() }
        }
        .errorAlert($model.errorMessage)
    }

    /// Clouds and hills on the hero screens, the plain gradient on list screens (n11).
    private var scenery: Bool {
        if model.engineMissing || model.showPermission { return true }
        if model.preview != nil { return false }
        switch model.destination {
        case .home: return model.stage != .report && model.stage != .review
        case .safeClean: return true
        case .explorer, .projects: return false
        case .protectedPaths: return protected.paths.isEmpty  // 11a keeps the scenery, the list does not
        }
    }

    @ViewBuilder
    private var detail: some View {
        if model.engineMissing {
            MissingEngine(disk: model.disk)
        } else if model.showPermission {
            PermissionView(model: model)
        } else if let preview = model.preview {
            PreviewPanel(model: model, state: preview)
        } else {
            destinationView
        }
    }

    @ViewBuilder
    private var destinationView: some View {
        switch model.destination {
        case .home:
            FlowView(model: model)
        case .safeClean:
            SafeCleanView(model: model)
        case .explorer:
            ExplorerView(model: explorer)
        case .projects:
            ProjectsView(model: projects, app: model)
        case .protectedPaths:
            ProtectedPathsView(model: protected)
        }
    }
}

struct Sidebar: View {
    @Bindable var model: AppModel

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            // Room for the window's traffic lights; the items start right below.
            Spacer().frame(height: 66)

            ForEach(Destination.main) { item(for: $0) }
            Spacer().frame(height: 26)
            ForEach(Destination.manage) { item(for: $0) }

            Spacer()

            Button {
                if model.accessLimited { model.showPermission = true }
            } label: {
                HStack(spacing: 8) {
                    Dot(color: model.accessLimited ? Palette.moderate : Palette.safe, size: 6)
                    Text(model.accessLimited ? "Accès partiel" : "Accès complet")
                }
                .font(.system(size: 12))
                .foregroundStyle(Palette.muted)
            }
            .buttonStyle(.plain)
            .help(model.accessLimited
                  ? "Accorder l'accès complet au disque"
                  : "Épurer a l'accès complet au disque")
        }
        .padding(.horizontal, 24)
        .padding(.bottom, 24)
        .frame(width: 220, alignment: .leading)
        .frame(maxHeight: .infinity, alignment: .top)
        .background(Palette.sidebar)
        .overlay(alignment: .trailing) {
            Rectangle().fill(Palette.sidebarEdge).frame(width: 1)
        }
    }

    private func item(for destination: Destination) -> some View {
        let active = model.destination == destination
        return Button {
            model.showPermission = false
            model.closePreview()
            model.destination = destination
        } label: {
            HStack(spacing: 10) {
                Dot(color: active ? Palette.accent : .clear, size: 5)
                Text(destination.title)
                    .font(.system(size: 14, weight: active ? .semibold : .regular))
                    .foregroundStyle(active ? Palette.ink : Palette.muted)
            }
            .frame(maxWidth: .infinity, minHeight: 34, alignment: .leading)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityAddTraits(active ? .isSelected : [])
    }
}
