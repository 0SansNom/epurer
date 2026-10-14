import AppKit
import SwiftUI

extension Color {
    /// One color per appearance: the design draws every board in light and dark.
    init(light: Color, dark: Color) {
        self.init(nsColor: NSColor(name: nil) { appearance in
            appearance.bestMatch(from: [.aqua, .darkAqua]) == .darkAqua ? NSColor(dark) : NSColor(light)
        })
    }

    init(hex: UInt32, opacity: Double = 1) {
        self.init(
            .sRGB,
            red: Double((hex >> 16) & 0xFF) / 255,
            green: Double((hex >> 8) & 0xFF) / 255,
            blue: Double(hex & 0xFF) / 255,
            opacity: opacity
        )
    }
}

enum Palette {
    static let ink = Color(light: Color(hex: 0x2B3752), dark: Color(hex: 0xEAF0FF))
    static let muted = Color(light: Color(hex: 0x4A566E), dark: Color(hex: 0xA9B6D3))
    static let secondary = muted
    /// Separators ("·", "···") and disabled labels.
    static let faint = Color(light: Color(hex: 0x5E6A82), dark: Color(hex: 0x8E9BB8))
    static let line = Color(light: Color(hex: 0x2B3752, opacity: 0.09), dark: .white.opacity(0.13))
    static let accent = Color(hex: 0x5B8DE8)
    static let link = Color(light: Color(hex: 0x2F5FC0), dark: Color(hex: 0x8FB4FF))
    /// "Supprimer" as a text action.
    static let deleteLabel = Color(light: Color(hex: 0xB53C47), dark: Color(hex: 0xFF9AA2))
    /// Text on amber notes ("Non monté", "déjà protégé").
    static let amberText = Color(light: Color(hex: 0x8A5A10), dark: Color(hex: 0xF0C987))
    /// Frosted fill inside rings and behind badges.
    static let glass = Color(light: .white.opacity(0.6), dark: Color(red: 44 / 255, green: 56 / 255, blue: 98 / 255, opacity: 0.62))
    /// Dialogs and floating bubbles.
    static let surface = Color(light: .white.opacity(0.96), dark: Color(red: 27 / 255, green: 36 / 255, blue: 68 / 255, opacity: 0.97))
    static let scrim = Color(light: Color(hex: 0x2B3752, opacity: 0.14), dark: .black.opacity(0.45))
    static let sidebar = Color(light: .white.opacity(0.55), dark: Color(red: 14 / 255, green: 22 / 255, blue: 46 / 255, opacity: 0.55))
    static let sidebarEdge = Color(light: Color(hex: 0x5B8DE8, opacity: 0.10), dark: .white.opacity(0.08))

    static let safe = Color(hex: 0x3FAE8A)
    static let moderate = Color(hex: 0xE0A94A)
    static let danger = Color(hex: 0xE0737B)

    static let ring = LinearGradient(
        stops: [
            .init(color: Color(hex: 0x8DB8F7), location: 0),
            .init(color: Color(hex: 0x8EDCCB), location: 0.55),
            .init(color: Color(hex: 0xCDE99A), location: 1),
        ],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    static let bar = LinearGradient(
        colors: [Color(hex: 0x8DB8F7), Color(hex: 0x8EDCCB)],
        startPoint: .leading,
        endPoint: .trailing
    )

    static let primary = LinearGradient(
        colors: [Color(hex: 0x4A80E2), Color(hex: 0x2F59B8)],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    static let destructive = LinearGradient(
        colors: [Color(hex: 0xD0505B), Color(hex: 0xB53C47)],
        startPoint: .topLeading,
        endPoint: .bottomTrailing
    )

    static func safety(_ s: Safety) -> Color {
        switch s {
        case .safe: return safe
        case .moderate: return moderate
        case .dangerous: return danger
        }
    }

    static func domain(_ domain: String) -> Color {
        switch domain {
        case "Mobile": return Color(hex: 0xE8A6DE)
        case "System": return Color(hex: 0xF3D08A)
        case "Frontend": return Color(hex: 0x8DB8F2)
        case "Backend": return Color(hex: 0xB9DD8A)
        case "DevOps": return Color(hex: 0xF4B8A6)
        case "Data/ML": return Color(hex: 0x9ADFCB)
        case "AI": return Color(hex: 0xA9DCEB)
        default: return Color(hex: 0xC9D0DC)
        }
    }

    /// Pill tint for a project type reported by `engine purge`.
    static func projectType(_ type: String) -> Color {
        switch type {
        case "Rust": return Color(hex: 0xB9DD8A)
        case "Node.js": return Color(hex: 0x8DB8F2)
        case "Swift", "CocoaPods/iOS", "Flutter": return Color(hex: 0xE8A6DE)
        case "Go": return Color(hex: 0xA9DCEB)
        case "Python": return Color(hex: 0x9ADFCB)
        case "Java (Maven)", "Java (Gradle)": return Color(hex: 0xF4B8A6)
        case "PHP", "C/C++ (CMake)": return Color(hex: 0xF3D08A)
        default: return accent.opacity(0.14)
        }
    }
}

/// The stylized sky behind every screen (note n11): clouds and hills where
/// the design wants them, the gradient alone on list screens.
struct AppBackground: View {
    /// Clouds, hills (and stars at night). Off on Estimation, Sélection,
    /// Explorer, Projets, Chemins protégés and their dialogs.
    var scenery: Bool
    @Environment(\.colorScheme) private var scheme

    var body: some View {
        let dark = scheme == .dark
        ZStack {
            LinearGradient(stops: stops(dark: dark), startPoint: .top, endPoint: .bottom)
            if scenery {
                Canvas { context, size in
                    // The boards draw the sky on a 1120 × 720 window.
                    context.scaleBy(x: size.width / 1120, y: size.height / 720)
                    clouds(&context, dark: dark)
                    hills(&context, dark: dark)
                    if dark { stars(&context) }
                }
            }
        }
        .ignoresSafeArea()
    }

    private func stops(dark: Bool) -> [Gradient.Stop] {
        switch (scenery, dark) {
        case (true, false):
            return [.init(color: Color(hex: 0xC9DFF8), location: 0), .init(color: Color(hex: 0xDCEAFB), location: 0.34),
                    .init(color: Color(hex: 0xF4EEF6), location: 0.66), .init(color: Color(hex: 0xFFE8D8), location: 1)]
        case (true, true):
            return [.init(color: Color(hex: 0x0F1A33), location: 0), .init(color: Color(hex: 0x1A2648), location: 0.38),
                    .init(color: Color(hex: 0x2A2B52), location: 0.68), .init(color: Color(hex: 0x3B2F52), location: 1)]
        case (false, false):
            return [.init(color: Color(hex: 0xE4EEFB), location: 0), .init(color: Color(hex: 0xF1F1FA), location: 0.55),
                    .init(color: Color(hex: 0xFBF1EB), location: 1)]
        case (false, true):
            return [.init(color: Color(hex: 0x121B35), location: 0), .init(color: Color(hex: 0x182140), location: 0.55),
                    .init(color: Color(hex: 0x211F3C), location: 1)]
        }
    }

    private func clouds(_ context: inout GraphicsContext, dark: Bool) {
        let fill = dark ? Color(hex: 0x6A7FC8) : .white
        let puffs: [(CGFloat, CGFloat, CGFloat, CGFloat, Double)] = [
            (420, 120, 150, 26, 0.85), (520, 146, 110, 20, 0.7), (900, 82, 130, 22, 0.8),
            (1010, 250, 100, 16, 0.55), (300, 300, 90, 14, 0.45),
        ]
        context.drawLayer { layer in
            layer.addFilter(.blur(radius: 16))
            for (x, y, rx, ry, alpha) in puffs {
                layer.fill(Path(ellipseIn: CGRect(x: x - rx, y: y - ry, width: rx * 2, height: ry * 2)),
                           with: .color(fill.opacity(alpha)))
            }
        }
    }

    private func hills(_ context: inout GraphicsContext, dark: Bool) {
        var back = Path()
        back.move(to: CGPoint(x: 0, y: 640))
        back.addCurve(to: CGPoint(x: 560, y: 636), control1: CGPoint(x: 180, y: 580), control2: CGPoint(x: 360, y: 600))
        back.addCurve(to: CGPoint(x: 1120, y: 628), control1: CGPoint(x: 760, y: 672), control2: CGPoint(x: 930, y: 600))
        back.addLine(to: CGPoint(x: 1120, y: 720)); back.addLine(to: CGPoint(x: 0, y: 720)); back.closeSubpath()
        var front = Path()
        front.move(to: CGPoint(x: 0, y: 676))
        front.addCurve(to: CGPoint(x: 680, y: 676), control1: CGPoint(x: 220, y: 640), control2: CGPoint(x: 440, y: 690))
        front.addCurve(to: CGPoint(x: 1120, y: 668), control1: CGPoint(x: 860, y: 666), control2: CGPoint(x: 990, y: 650))
        front.addLine(to: CGPoint(x: 1120, y: 720)); front.addLine(to: CGPoint(x: 0, y: 720)); front.closeSubpath()

        func shade(_ top: UInt32, _ bottom: UInt32, _ alpha: Double, from y: CGFloat) -> GraphicsContext.Shading {
            .linearGradient(Gradient(colors: [Color(hex: top, opacity: alpha), Color(hex: bottom, opacity: alpha)]),
                            startPoint: CGPoint(x: 0, y: y), endPoint: CGPoint(x: 0, y: 720))
        }
        context.drawLayer { layer in
            layer.addFilter(.blur(radius: 3))
            layer.fill(back, with: dark ? shade(0x1B3742, 0x152D38, 0.85, from: 580) : shade(0xBFE3D3, 0xDDF0DA, 0.85, from: 580))
            layer.fill(front, with: dark ? shade(0x1F3F49, 0x193640, 0.95, from: 640) : shade(0xCFEBDD, 0xE6F4E2, 0.95, from: 640))
        }
    }

    private func stars(_ context: inout GraphicsContext) {
        let stars: [(CGFloat, CGFloat, CGFloat, Double)] = [
            (877, 144, 1.3, 0.7), (997, 285, 0.8, 0.5), (1034, 141, 0.8, 0.35), (355, 204, 1.6, 0.35),
            (629, 292, 0.8, 0.7), (495, 20, 1, 0.5), (526, 107, 1.6, 0.35), (1020, 50, 1, 0.7),
            (872, 241, 1, 0.35), (241, 16, 1, 0.35), (409, 99, 1.3, 0.5), (443, 290, 1, 0.35),
            (947, 114, 1.6, 0.5), (262, 198, 1.6, 0.35), (389, 149, 0.8, 0.5), (548, 322, 0.8, 0.7),
            (934, 187, 0.8, 0.5), (603, 170, 1.6, 0.7), (563, 108, 1.6, 0.5), (961, 104, 0.8, 0.5),
            (802, 228, 1.3, 0.5), (832, 18, 1.6, 0.35), (1013, 139, 1.6, 0.5), (764, 195, 1.3, 0.5),
            (350, 315, 1.3, 0.5), (277, 235, 0.8, 0.35), (588, 276, 1.3, 0.35), (588, 155, 0.8, 0.5),
            (942, 176, 1.3, 0.35), (1057, 54, 1, 0.7), (977, 38, 0.8, 0.7),
        ]
        for (x, y, r, alpha) in stars {
            context.fill(Path(ellipseIn: CGRect(x: x - r, y: y - r, width: r * 2, height: r * 2)), with: .color(.white.opacity(alpha)))
        }
    }
}

// MARK: - Buttons

struct PrimaryPillStyle: ButtonStyle {
    var height: CGFloat = 50
    var horizontalPadding: CGFloat = 40
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 15, weight: .semibold))
            .foregroundStyle(.white)
            .padding(.horizontal, horizontalPadding)
            .frame(height: height)
            .background(Capsule().fill(Palette.primary))
            .shadow(color: Color(hex: 0x5B8DE8, opacity: isEnabled ? 0.24 : 0), radius: 12, y: 10)
            .opacity(isEnabled ? (configuration.isPressed ? 0.85 : 1) : 0.4)
            .scaleEffect(configuration.isPressed ? 0.98 : 1)
            .contentShape(Capsule())
    }
}

/// The big round "Scanner" button on the home screen, with its two halos.
struct ScanOrbStyle: ButtonStyle {
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 17, weight: .bold))
            .foregroundStyle(isEnabled ? .white : Palette.faint)
            .frame(width: 108, height: 108)
            .background(Circle().fill(isEnabled ? AnyShapeStyle(Palette.primary) : AnyShapeStyle(Palette.ink.opacity(0.07))))
            .shadow(color: Color(hex: 0x5B8DE8, opacity: isEnabled ? 0.26 : 0), radius: 16, y: 14)
            .background(Circle().fill(Color(hex: 0x8DB8F7, opacity: isEnabled ? 0.14 : 0)).padding(-12))
            .scaleEffect(configuration.isPressed ? 0.96 : 1)
            .animation(.spring(duration: 0.25), value: configuration.isPressed)
            .contentShape(Circle())
    }
}

/// Primary pill with a soft halo (Confirm, Nettoyage sûr).
struct HaloPillStyle: ButtonStyle {
    var height: CGFloat = 52
    var horizontalPadding: CGFloat = 48

    func makeBody(configuration: Configuration) -> some View {
        PrimaryPillStyle(height: height, horizontalPadding: horizontalPadding)
            .makeBody(configuration: configuration)
            .background(Capsule().fill(Color(hex: 0x8DB8F7, opacity: 0.14)).padding(-8))
    }
}

/// Coral pill for the one destructive action ("Supprimer").
struct DangerPillStyle: ButtonStyle {
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 14, weight: .semibold))
            .foregroundStyle(.white)
            .padding(.horizontal, 26)
            .frame(height: 42)
            .background(Capsule().fill(Palette.destructive))
            .shadow(color: Color(hex: 0xBE434E, opacity: isEnabled ? 0.30 : 0), radius: 9, y: 8)
            .opacity(isEnabled ? (configuration.isPressed ? 0.85 : 1) : 0.4)
            .contentShape(Capsule())
    }
}

/// White pill with a blue outline ("Choisir…").
struct SecondaryPillStyle: ButtonStyle {
    var fill: Color = Palette.glass
    var height: CGFloat = 44
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 14, weight: .semibold))
            .foregroundStyle(isEnabled ? Palette.link : Palette.faint)
            .padding(.horizontal, 24)
            .frame(height: height)
            .background(Capsule().fill(isEnabled ? fill : Palette.ink.opacity(0.04)))
            .overlay(Capsule().strokeBorder(isEnabled ? Palette.accent.opacity(0.35) : Palette.ink.opacity(0.10)))
            .opacity(configuration.isPressed ? 0.7 : 1)
            .contentShape(Capsule())
    }
}

/// Blue text action ("Rescanner", "Réglages", "Révéler").
struct LinkStyle: ButtonStyle {
    var color: Color = Palette.link
    var size: CGFloat = 13

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: size))
            .foregroundStyle(color)
            .frame(minHeight: 30)
            .contentShape(Rectangle())
            .opacity(configuration.isPressed ? 0.6 : 1)
    }
}

struct QuietStyle: ButtonStyle {
    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 13))
            .foregroundStyle(Palette.muted)
            .frame(minHeight: 36)
            .contentShape(Rectangle())
            .opacity(configuration.isPressed ? 0.6 : 1)
    }
}

// MARK: - Small pieces

struct Dot: View {
    var color: Color
    var size: CGFloat = 7

    var body: some View {
        Circle().fill(color).frame(width: size, height: size)
    }
}

struct DotLabel: View {
    var color: Color
    var text: String

    var body: some View {
        HStack(spacing: 7) {
            Dot(color: color)
            Text(text)
        }
    }
}

struct BigNumber: View {
    var text: String
    var size: CGFloat
    var weight: Font.Weight = .ultraLight

    var body: some View {
        Text(text)
            .font(.system(size: size, weight: weight))
            .tracking(-size * 0.045)
            .monospacedDigit()
            .lineLimit(1)
            .minimumScaleFactor(0.5)
    }
}

struct MonoPath: View {
    var text: String

    var body: some View {
        Text(text)
            .font(.system(size: 11, design: .monospaced))
            .foregroundStyle(Palette.muted)
            .lineLimit(1)
            .truncationMode(.middle)
            .frame(width: 420)
    }
}

/// The rounded cards from the "États" board: a 132 pt ring holding a
/// mascot or glyph, a title, a line of text and one action.
struct StateCard<Glyph: View, Action: View>: View {
    enum Ring { case dashed, full }

    var ring: Ring
    var title: String
    var text: String
    /// Off on 11a, where the ring sits on the page without the card behind it.
    var card = true
    @ViewBuilder var glyph: () -> Glyph
    @ViewBuilder var action: () -> Action

    var body: some View {
        VStack(spacing: 0) {
            ZStack {
                switch ring {
                case .dashed:
                    Circle().stroke(Palette.accent.opacity(0.12), style: StrokeStyle(lineWidth: 7, dash: [3, 7]))
                case .full:
                    Circle().stroke(Palette.ring, lineWidth: 7)
                }
                glyph()
            }
            .frame(width: 118, height: 118)
            .padding(7)

            Text(title)
                .font(.system(size: 17, weight: .semibold))
                .padding(.top, 16)
            Text(text)
                .font(.system(size: 12))
                .foregroundStyle(Palette.secondary)
                .multilineTextAlignment(.center)
                .padding(.top, 4)
            action()
                .padding(.top, 14)
        }
        .padding(.top, 46)
        .padding(.horizontal, 20)
        .frame(width: 330, height: 308, alignment: .top)
        .background(RoundedRectangle(cornerRadius: 24).fill(Palette.accent.opacity(card ? 0.035 : 0)))
        .overlay(RoundedRectangle(cornerRadius: 24).strokeBorder(Palette.ink.opacity(card ? 0.07 : 0)))
    }
}

/// "Analyse en cours": spinning partial ring with skeleton lines.
struct AnalysingCard: View {
    var text = String(localized: "Estimation en cours…")
    /// Off on work screens (Explorer, Projets), where Pépin never shows.
    var mascot = true
    var onCancel: (() -> Void)?

    @State private var spin = false

    var body: some View {
        StateCard(ring: .dashed, title: String(localized: "Analyse en cours"), text: text) {
            ZStack {
                Circle().stroke(Palette.accent.opacity(0.12), lineWidth: 7)
                Circle().trim(from: 0, to: 0.44)
                    .stroke(Palette.ring, style: StrokeStyle(lineWidth: 7, lineCap: .round))
                    .rotationEffect(.degrees(spin ? 270 : -90))
                    .animation(.linear(duration: 1.6).repeatForever(autoreverses: false), value: spin)
                    .onAppear { spin = true }
                if mascot {
                    Mascot(mood: .neutre, size: 84)
                }
            }
        } action: {
            if let onCancel {
                Button("Annuler", action: onCancel)
                    .buttonStyle(LinkStyle(color: Palette.accent))
                    .keyboardShortcut(.cancelAction)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

/// "Prêt à scanner": sleepy Mousse in a dashed ring, before any analysis.
struct ReadyCard: View {
    var text = String(localized: "Aucune analyse pour l’instant.")
    var onScan: () -> Void

    var body: some View {
        StateCard(ring: .dashed, title: String(localized: "Prêt à scanner"), text: text) {
            Mascot(mood: .endormi, size: 84)
        } action: {
            Button("Scanner", action: onScan)
                .buttonStyle(PrimaryPillStyle(height: 34, horizontalPadding: 22))
                .keyboardShortcut(.defaultAction)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

/// "Tout est propre": content Pépin with a green check badge.
struct DoneGlyph: View {
    var size: CGFloat = 84
    var badgeSize: CGFloat = 34

    var body: some View {
        BadgedMascot(mood: .content, size: size, badgeSize: badgeSize) {
            Image(systemName: "checkmark")
                .font(.system(size: badgeSize * 0.38, weight: .heavy))
                .foregroundStyle(Palette.safe)
        }
        .accessibilityLabel("Tout est propre")
    }
}

/// "Accès limité": embarrassed Pépin with an amber lock badge.
struct LockedGlyph: View {
    var size: CGFloat = 84
    var badgeSize: CGFloat = 34

    var body: some View {
        BadgedMascot(mood: .gene, size: size, badgeSize: badgeSize) {
            Image(systemName: "lock.fill")
                .font(.system(size: badgeSize * 0.45))
                .foregroundStyle(Palette.moderate)
        }
        .accessibilityLabel("Accès limité")
    }
}

extension View {
    /// The app's one error alert, shown while `message` is set.
    func errorAlert(_ message: Binding<String?>) -> some View {
        alert("Épurer", isPresented: Binding(
            get: { message.wrappedValue != nil },
            set: { if !$0 { message.wrappedValue = nil } }
        )) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(message.wrappedValue ?? "")
        }
    }
}

// MARK: - Formatting

enum Fmt {
    /// Translated unit names: "Go" in French, "GB" in English.
    private static let units = [
        String(localized: "o"), String(localized: "Ko"), String(localized: "Mo"),
        String(localized: "Go"), String(localized: "To"),
    ]

    /// Decimal (Finder-style) sizes in the app's language: "12,4 Go", "12.4 GB".
    static func bytes(_ count: Int64) -> String {
        var value = Double(max(count, 0))
        var unit = 0
        while value >= 1000, unit < units.count - 1 {
            value /= 1000
            unit += 1
        }
        let nf = (unit == 0 || value >= 100) ? whole : oneDecimal
        return "\(nf.string(from: value as NSNumber) ?? "0") \(units[unit])"
    }

    static func count(_ n: Int) -> String {
        grouped.string(from: n as NSNumber) ?? "\(n)"
    }

    private static func formatter(fractionDigits: Int, style: NumberFormatter.Style = .none) -> NumberFormatter {
        let nf = NumberFormatter()
        nf.numberStyle = style
        nf.minimumFractionDigits = 0
        nf.maximumFractionDigits = fractionDigits
        return nf
    }

    private static let whole = formatter(fractionDigits: 0)
    private static let oneDecimal = formatter(fractionDigits: 1)
    private static let grouped = formatter(fractionDigits: 0, style: .decimal)
}
