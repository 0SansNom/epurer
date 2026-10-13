import AppKit
import SwiftUI

@main
struct EpurerApp: App {
    var body: some Scene {
        WindowGroup {
            ContentView()
                .frame(minWidth: 960, minHeight: 640)
                .background(UnifiedTitlebar())
        }
        .windowStyle(.hiddenTitleBar)
        .defaultSize(width: 1120, height: 720)
        .commands {
            CommandGroup(replacing: .newItem) {}
        }
    }
}

/// An empty, transparent unified toolbar: macOS then insets the window
/// buttons like its own sidebar apps (Notes, Mail, Finder) instead of
/// pinning them to the corner of a plain title bar.
private struct UnifiedTitlebar: NSViewRepresentable {
    func makeNSView(context: Context) -> NSView {
        let view = NSView()
        DispatchQueue.main.async {
            guard let window = view.window else { return }
            window.titleVisibility = .hidden
            window.titlebarAppearsTransparent = true
            window.styleMask.insert(.fullSizeContentView)
            window.toolbar = NSToolbar(identifier: "epurer.main")
            window.toolbarStyle = .unified
            window.titlebarSeparatorStyle = .none
        }
        return view
    }

    func updateNSView(_ nsView: NSView, context: Context) {}
}
