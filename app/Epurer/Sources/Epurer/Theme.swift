import SwiftUI

extension Color {
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
    static let ink = Color(hex: 0x2B3752)
    static let muted = Color(hex: 0x66728A)
    static let secondary = Color(hex: 0x5A667E)
    static let line = Color(hex: 0x2B3752, opacity: 0.09)
    static let accent = Color(hex: 0x5B8DE8)
    static let link = Color(hex: 0x3F6FD0)
    static let ground = Color(hex: 0xFCFBFF)

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
        colors: [Color(hex: 0x7FAEF2), Color(hex: 0x5B8DE8)],
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

/// Pale peach / sky / pink washes over a near-white ground.
struct AppBackground: View {
    var body: some View {
        GeometryReader { geo in
            let w = geo.size.width
            ZStack {
                Palette.ground
                wash(Color(red: 1, green: 233 / 255, blue: 196 / 255, opacity: 0.6), at: UnitPoint(x: 0.08, y: 0), radius: w * 0.45)
                wash(Color(red: 196 / 255, green: 224 / 255, blue: 1, opacity: 0.55), at: UnitPoint(x: 0.94, y: 0.06), radius: w * 0.45)
                wash(Color(red: 1, green: 210 / 255, blue: 238 / 255, opacity: 0.4), at: UnitPoint(x: 0.62, y: 1.04), radius: w * 0.4)
            }
        }
        .ignoresSafeArea()
    }

    private func wash(_ color: Color, at center: UnitPoint, radius: CGFloat) -> some View {
        RadialGradient(colors: [color, color.opacity(0)], center: center, startRadius: 0, endRadius: radius)
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
            .foregroundStyle(isEnabled ? .white : Color(hex: 0x8A94A8))
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
            .background(Capsule().fill(LinearGradient(
                colors: [Color(hex: 0xEE9AA0), Color(hex: 0xE0737B)],
                startPoint: .topLeading,
                endPoint: .bottomTrailing
            )))
            .shadow(color: Color(hex: 0xE0737B, opacity: isEnabled ? 0.28 : 0), radius: 9, y: 8)
            .opacity(isEnabled ? (configuration.isPressed ? 0.85 : 1) : 0.4)
            .contentShape(Capsule())
    }
}

/// White pill with a blue outline ("Choisir…").
struct SecondaryPillStyle: ButtonStyle {
    var fill: Color = .white.opacity(0.7)
    var height: CGFloat = 44
    @Environment(\.isEnabled) private var isEnabled

    func makeBody(configuration: Configuration) -> some View {
        configuration.label
            .font(.system(size: 14, weight: .semibold))
            .foregroundStyle(isEnabled ? Palette.link : Color(hex: 0x8A94A8))
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
        .background(RoundedRectangle(cornerRadius: 24).fill(Palette.accent.opacity(0.035)))
        .overlay(RoundedRectangle(cornerRadius: 24).strokeBorder(Palette.ink.opacity(0.07)))
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
