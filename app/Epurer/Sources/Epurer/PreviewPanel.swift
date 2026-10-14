import SwiftUI

/// 10b · "Prévisualiser": what the next "Nettoyer" would delete, checked by
/// an engine dry run. Nothing is removed from here.
/// 10c · In confirm mode the same list is the last step before deleting.
struct PreviewPanel: View {
    @Bindable var model: AppModel
    var state: PreviewState

    private static let noteLimit = 3

    private var confirming: Bool { state.request.mode == .confirm }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 8) {
                Dot(color: confirming ? Palette.danger : Color(hex: 0x8DB8F7), size: 6)
                Text(confirming ? "Suppression définitive" : "Prévisualisation · rien n’est supprimé")
            }
            .font(.system(size: 12))
            .foregroundStyle(Palette.muted)
            .frame(height: 28)

            if let results = state.results {
                content(results)
            } else {
                AnalysingCard(text: String(localized: "Vérification de chaque élément…"), mascot: false) { model.closePreview() }
            }
        }
        .padding(EdgeInsets(top: 32, leading: 72, bottom: 0, trailing: 72))
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }

    @ViewBuilder
    private func content(_ results: [CleanResult]) -> some View {
        let request = state.request
        let byPath = Dictionary(request.entries.map { ($0.target.path, $0) }, uniquingKeysWith: { a, _ in a })
        let ok = results.filter(\.success).compactMap { byPath[$0.path] }
        let total = results.filter(\.success).reduce(0) { $0 + $1.bytesFreed }
        let failed = results.filter { $0.status == "failed" }
        let lockedPaths = results.filter { $0.status == "left_in_place" }.map(\.path)
        let lockedCount = lockedPaths.count + request.locked.count

        if confirming, let question = request.question {
            Text(question)
                .font(.system(size: 15))
                .foregroundStyle(Palette.muted)
                .padding(.top, 14)
        }
        BigNumber(text: Fmt.bytes(total), size: 56, weight: .ultraLight)
            .padding(.top, confirming && request.question != nil ? 6 : 14)
        Text(subtitle(ok, request))
            .foregroundStyle(Palette.muted)
            .padding(.top, 6)

        ScrollView {
            LazyVStack(spacing: 0) {
                ForEach(ok) { entry in
                    PreviewRow(entry: entry)
                }
            }
        }
        .padding(.top, 20)

        notes(request: request, failed: failed, lockedCount: lockedCount)

        HStack {
            Spacer()
            if confirming {
                Button("Annuler") { model.closePreview() }
                    .buttonStyle(QuietStyle())
                    .keyboardShortcut(.cancelAction)
                    .padding(.trailing, 16)
            } else {
                Button("Fermer") { model.closePreview() }
                    .buttonStyle(QuietStyle())
                    .keyboardShortcut(.cancelAction)
                    .padding(.trailing, 16)
            }
            Button(request.actionTitle) {
                if request.confirmFirst && !confirming {
                    model.setPreviewMode(.confirm)
                } else {
                    model.closePreview()
                    request.onClean()
                }
            }
            .buttonStyle(PrimaryPillStyle(height: 48, horizontalPadding: 32))
            .disabled(ok.isEmpty)
            .keyboardShortcut(.defaultAction)
        }
        .frame(height: 84)
        .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
    }

    private func subtitle(_ ok: [PreviewEntry], _ request: PreviewRequest) -> String {
        let n = ok.count
        var parts = [
            confirming ? String(localized: "seront libérés") : String(localized: "seraient libérés"),
            request.folders ? String(localized: "\(n) dossiers") : String(localized: "\(n) éléments"),
        ]
        if request.countsProjects {
            let projects = Set(ok.compactMap(\.project)).count
            parts.append(String(localized: "\(projects) projets"))
        }
        return parts.joined(separator: " · ")
    }

    /// One line, e.g. "Laissés de côté : site-portfolio (non sélectionné), 1 dossier verrouillé".
    @ViewBuilder
    private func notes(request: PreviewRequest, failed: [CleanResult], lockedCount: Int) -> some View {
        if let text = Self.leftAside(request: request, failed: failed.count, locked: lockedCount) {
            note(Palette.moderate, text)
                .padding(.top, 12)
                .frame(maxWidth: .infinity, alignment: .leading)
                .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
                .padding(.top, 14)
        }
    }

    static func leftAside(request: PreviewRequest, failed: Int, locked: Int) -> String? {
        var seen = Set<String>()
        let names = request.skipped.map { $0.project ?? $0.target.label }.filter { seen.insert($0).inserted }
        var parts: [String] = []
        if !names.isEmpty {
            let shown = names.prefix(noteLimit).joined(separator: ", ")
            let rest = names.count - noteLimit
            let list = rest > 0 ? String(localized: "\(shown) et \(rest) autres") : shown
            let status = names.count > 1 ? String(localized: "non sélectionnés") : String(localized: "non sélectionné")
            parts.append("\(list) (\(status))")
        }
        if locked > 0 {
            parts.append(request.folders ? String(localized: "\(locked) dossiers verrouillés") : String(localized: "\(locked) éléments verrouillés"))
        }
        if failed > 0 {
            parts.append(String(localized: "\(failed) échecs prévus"))
        }
        return parts.isEmpty ? nil : String(localized: "Laissés de côté : \(parts.joined(separator: ", "))")
    }

    private func note(_ color: Color, _ text: String) -> some View {
        HStack(spacing: 8) {
            Dot(color: color, size: 6)
            Text(text).lineLimit(1).truncationMode(.middle)
        }
        .font(.system(size: 12))
        .foregroundStyle(Palette.muted)
    }
}

struct PreviewRow: View {
    var entry: PreviewEntry

    private static let relative: RelativeDateTimeFormatter = {
        let f = RelativeDateTimeFormatter()
        f.unitsStyle = .abbreviated
        return f
    }()

    private var modified: String {
        guard let attrs = try? FileManager.default.attributesOfItem(atPath: entry.target.path),
              let date = attrs[.modificationDate] as? Date
        else { return "" }
        return Self.relative.localizedString(for: date, relativeTo: Date())
    }

    var body: some View {
        HStack(spacing: 16) {
            Group {
                if let project = entry.project {
                    HStack(spacing: 4) {
                        Text(project)
                        Text("/").foregroundStyle(Palette.muted)
                        Text(entry.target.label).font(.system(size: 13, design: .monospaced))
                    }
                } else {
                    Text(entry.target.label)
                }
            }
            .font(.system(size: 14))
            .lineLimit(1)
            .truncationMode(.middle)
            .frame(maxWidth: .infinity, alignment: .leading)
            .help(displayPath(entry.target.path))

            Text(modified)
                .font(.system(size: 12))
                .foregroundStyle(Palette.muted)
                .frame(width: 110, alignment: .trailing)
            Text(Fmt.bytes(entry.target.sizeBytes))
                .font(.system(size: 14))
                .monospacedDigit()
                .frame(width: 70, alignment: .trailing)
        }
        .frame(height: 38)
        .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
    }
}
