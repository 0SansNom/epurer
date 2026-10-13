import AppKit
import SwiftUI

// MARK: 8 · Nettoyage sûr

/// Safe items only, cleaned in one gesture.
struct SafeCleanView: View {
    @Bindable var model: AppModel

    var body: some View {
        Group {
            switch model.safeState {
            case .idle:
                ReadyCard { model.scanSafe() }
            case .loading:
                detecting
            case .ready where model.safeTargets.isEmpty:
                StateCard(ring: .full, title: String(localized: "Tout est propre"), text: String(localized: "Rien à nettoyer.")) {
                    DoneGlyph()
                } action: {
                    Button("Rescanner") { model.scanSafe() }
                        .buttonStyle(LinkStyle(color: Palette.accent))
                }
                .frame(maxWidth: .infinity, maxHeight: .infinity)
            case .ready:
                summary
            }
        }
        .task { if model.safeState == .idle && !model.safeCancelled { model.scanSafe() } }
    }

    /// 8a · Detection at the Prudent level, before the first summary.
    private var detecting: some View {
        RingProgress(
            fraction: nil,
            mood: .neutre,
            tally: model.safeTally,
            caption: String(localized: "Éléments sûrs · niveau Prudent"),
            label: String(localized: "Détection des éléments sûrs en cours"),
            buttonTitle: String(localized: "Annuler"),
            action: model.cancelSafe
        )
    }

    private var summary: some View {
        VStack(spacing: 0) {
            Mascot(mood: .content, size: 92)
            Text("Nettoyage sûr")
                .font(.system(size: 15))
                .foregroundStyle(Palette.muted)
                .padding(.top, 8)
            BigNumber(text: (model.accessLimited ? "≥ " : "") + Fmt.bytes(model.safeBytes), size: 104)
                .padding(.top, 8)
            Text(model.includesTrash(model.safeTargets) ? "\(model.safeTargets.count) éléments sûrs · Corbeille comprise" : "\(model.safeTargets.count) éléments sûrs")
                .foregroundStyle(Palette.muted)
                .padding(.top, 10)

            // "Corbeille 1,9 Go · Caches utilisateur 2,6 Go · …", wrapping.
            FlowLayout(spacing: 10) {
                ForEach(Array(model.safeGroups.prefix(12).enumerated()), id: \.element.name) { index, group in
                    if index > 0 {
                        Text("·").foregroundStyle(Palette.ink.opacity(0.25))
                    }
                    HStack(spacing: 4) {
                        Text(group.name)
                        Text(Fmt.bytes(group.bytes)).foregroundStyle(Palette.muted)
                    }
                }
            }
            .frame(maxWidth: 760)
            .padding(.top, 26)

            Button("Nettoyer en un geste") { model.cleanSafe() }
                .buttonStyle(HaloPillStyle(height: 56, horizontalPadding: 52))
                .font(.system(size: 16, weight: .semibold))
                .keyboardShortcut(.defaultAction)
                .padding(.top, 34)
            Button("Prévisualiser") { model.previewSafe() }
                .buttonStyle(LinkStyle())
                .fontWeight(.semibold)
                .padding(.top, 12)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

/// Wraps chips onto as many centered lines as needed.
struct FlowLayout: Layout {
    var spacing: CGFloat = 8

    func sizeThatFits(proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) -> CGSize {
        let rows = rows(for: subviews, width: proposal.width ?? .infinity)
        let height = rows.map(\.height).reduce(0, +) + spacing * CGFloat(max(rows.count - 1, 0))
        let width = rows.map(\.width).max() ?? 0
        return CGSize(width: proposal.width ?? width, height: height)
    }

    func placeSubviews(in bounds: CGRect, proposal: ProposedViewSize, subviews: Subviews, cache: inout ()) {
        var y = bounds.minY
        for row in rows(for: subviews, width: bounds.width) {
            var x = bounds.minX + (bounds.width - row.width) / 2
            for index in row.indices {
                let size = subviews[index].sizeThatFits(.unspecified)
                subviews[index].place(at: CGPoint(x: x, y: y), proposal: ProposedViewSize(size))
                x += size.width + spacing
            }
            y += row.height + spacing
        }
    }

    private struct Row {
        var indices: [Int] = []
        var width: CGFloat = 0
        var height: CGFloat = 0
    }

    private func rows(for subviews: Subviews, width: CGFloat) -> [Row] {
        var rows: [Row] = []
        var current = Row()
        for (index, view) in subviews.enumerated() {
            let size = view.sizeThatFits(.unspecified)
            let extra = current.indices.isEmpty ? size.width : current.width + spacing + size.width
            if extra > width, !current.indices.isEmpty {
                rows.append(current)
                current = Row()
            }
            current.width = current.indices.isEmpty ? size.width : current.width + spacing + size.width
            current.height = max(current.height, size.height)
            current.indices.append(index)
        }
        if !current.indices.isEmpty { rows.append(current) }
        return rows
    }
}

// MARK: 13 · Accès complet au disque

/// 13 → 13a → 13b: talk about the benefit, show which switch to find,
/// wait for it, confirm. "Continuer avec un accès partiel" is always there.
struct PermissionView: View {
    enum Phase { case intro, waiting, done }

    @Bindable var model: AppModel
    @State private var phase = Phase.intro
    @State private var spin = false

    var body: some View {
        VStack(spacing: 0) {
            glyph
            Text(title)
                .font(.system(size: 30, weight: .light))
                .tracking(-1)
                .padding(.top, 24)
            Text(subtitle)
                .font(.system(size: 14))
                .foregroundStyle(Palette.secondary)
                .multilineTextAlignment(.center)
                .lineSpacing(4)
                .frame(width: 400)
                .padding(.top, 8)

            SettingsRowHint(on: phase == .done)
                .padding(.top, 26)

            switch phase {
            case .intro:
                Text("Réglages Système › Confidentialité et sécurité › Accès complet au disque")
                    .font(.system(size: 12))
                    .foregroundStyle(Palette.secondary)
                    .padding(.top, 12)
                Button("Ouvrir les Réglages") {
                    model.openFullDiskAccessSettings()
                    phase = .waiting
                }
                .buttonStyle(PrimaryPillStyle())
                .keyboardShortcut(.defaultAction)
                .padding(.top, 28)
                partialButton
                Text("Aucun mot de passe administrateur. Rien n’est supprimé sans votre confirmation.")
                    .font(.system(size: 11))
                    .foregroundStyle(Color(hex: 0x7A869C))
                    .padding(.top, 10)
            case .waiting:
                Button("Rouvrir les Réglages") { model.openFullDiskAccessSettings() }
                    .buttonStyle(SecondaryPillStyle(fill: Palette.accent.opacity(0.06)))
                    .padding(.top, 28)
                partialButton
            case .done:
                Button("Scanner") { model.scan() }
                    .buttonStyle(PrimaryPillStyle())
                    .keyboardShortcut(.defaultAction)
                    .padding(.top, 28)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
        .animation(.easeInOut(duration: 0.2), value: phase)
        .onChange(of: model.accessLimited) { _, limited in
            if !limited { phase = .done }
        }
        .task(id: phase) {
            // 13a · Poll until macOS reports the grant.
            guard phase == .waiting else { return }
            while !Task.isCancelled {
                await model.refreshAccess()
                if !model.accessLimited {
                    phase = .done
                    return
                }
                try? await Task.sleep(for: .seconds(2))
            }
        }
    }

    private var title: String {
        switch phase {
        case .intro: return String(localized: "Voir tous vos caches")
        case .waiting: return String(localized: "En attente de l’activation")
        case .done: return String(localized: "Accès activé")
        }
    }

    private var subtitle: String {
        switch phase {
        case .intro: return String(localized: "Autorisez l’accès complet au disque pour obtenir des tailles exactes. Sans lui, Épurer affiche des minimums.")
        case .waiting: return String(localized: "Activez Épurer dans les Réglages. macOS vous proposera ensuite de le rouvrir.")
        case .done: return String(localized: "Épurer peut maintenant mesurer tous vos caches avec précision.")
        }
    }

    @ViewBuilder
    private var glyph: some View {
        switch phase {
        case .intro:
            LockedGlyph(size: 112, badgeSize: 38)
        case .waiting:
            ZStack {
                Circle().stroke(Palette.accent.opacity(0.14), style: StrokeStyle(lineWidth: 8, dash: [3, 8]))
                Circle().trim(from: 0, to: 0.2)
                    .stroke(Color(hex: 0x8DB8F7), style: StrokeStyle(lineWidth: 8, lineCap: .round))
                    .rotationEffect(.degrees(spin ? 270 : -90))
                    .animation(.linear(duration: 1.6).repeatForever(autoreverses: false), value: spin)
                    .onAppear { spin = true }
                Mascot(mood: .neutre, size: 92)
            }
            .frame(width: 140, height: 140)
            .accessibilityLabel("En attente de l’activation")
        case .done:
            DoneGlyph(size: 112, badgeSize: 38)
        }
    }

    private var partialButton: some View {
        Button("Continuer avec un accès partiel") { model.showPermission = false }
            .buttonStyle(LinkStyle())
            .fontWeight(.semibold)
            .keyboardShortcut(.cancelAction)
            .padding(.top, 8)
    }
}

/// Look-alike of the System Settings row, so people know which switch to find.
struct SettingsRowHint: View {
    var on: Bool

    var body: some View {
        HStack(spacing: 12) {
            RoundedRectangle(cornerRadius: 8)
                .fill(LinearGradient(colors: [Color(hex: 0xE6F1FF), Color(hex: 0x8EDCCB)], startPoint: .topLeading, endPoint: .bottomTrailing))
                .frame(width: 28, height: 28)
            Text("Épurer").font(.system(size: 15))
            Spacer()
            Capsule()
                .fill(on ? Palette.accent : Palette.ink.opacity(0.18))
                .frame(width: 42, height: 24)
                .overlay(alignment: on ? .trailing : .leading) {
                    Circle().fill(.white)
                        .frame(width: 20, height: 20)
                        .shadow(color: Palette.ink.opacity(0.25), radius: 1.5, y: 1)
                        .padding(2)
                }
        }
        .padding(.horizontal, 16)
        .frame(width: 360, height: 52)
        .background(RoundedRectangle(cornerRadius: 16).fill(Palette.accent.opacity(0.05)))
        .overlay(RoundedRectangle(cornerRadius: 16).strokeBorder(Palette.ink.opacity(0.08)))
        .accessibilityElement(children: .ignore)
        .accessibilityLabel(on ? "Épurer est activé dans l’accès complet au disque" : "Interrupteur Épurer à activer dans l’accès complet au disque")
    }
}
