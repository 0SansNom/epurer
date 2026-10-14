import SwiftUI

/// Pépin, the only mascot: six moods, drawn by the designer as the
/// "Pepin-<mood>" images in Assets.xcassets (128 pt, @1x to @3x).
/// Never shown where things get deleted.
struct Mascot: View {
    enum Mood: String { case content, neutre, endormi, surpris, concentre, gene }

    var mood: Mood = .content
    var size: CGFloat = 112

    var body: some View {
        Image("Pepin-\(mood.rawValue)")
            .resizable()
            .interpolation(.high)
            .scaledToFit()
            .frame(width: size, height: size)
            .accessibilityLabel("Pépin, mascotte d'Épurer")
    }
}

/// Mascot with a small round badge at its lower right (✓ or lock).
struct BadgedMascot<Badge: View>: View {
    var mood: Mascot.Mood
    var size: CGFloat
    var badgeSize: CGFloat = 34
    @ViewBuilder var badge: () -> Badge

    var body: some View {
        Mascot(mood: mood, size: size)
            .overlay(alignment: .bottomTrailing) {
                badge()
                    .frame(width: badgeSize, height: badgeSize)
                    .background(Circle().fill(Palette.glass))
                    .offset(x: badgeSize * 0.3, y: badgeSize * 0.1)
            }
    }
}

/// The 320 pt gradient ring used on home, scan and clean screens.
struct ProgressRing<Center: View>: View {
    /// nil = indeterminate (a spinning arc).
    var fraction: Double?
    var diameter: CGFloat = 320
    var lineWidth: CGFloat = 16
    @ViewBuilder var center: () -> Center

    @State private var spin = false

    var body: some View {
        ZStack {
            Circle().fill(Palette.glass)
            Circle().stroke(Palette.accent.opacity(0.12), lineWidth: lineWidth)
            if let fraction {
                Circle()
                    .trim(from: 0, to: max(0.001, min(fraction, 1)))
                    .stroke(Palette.ring, style: StrokeStyle(lineWidth: lineWidth, lineCap: .round))
                    .rotationEffect(.degrees(-90))
                    .animation(.easeOut(duration: 0.4), value: fraction)
            } else {
                Circle()
                    .trim(from: 0, to: 0.38)
                    .stroke(Palette.ring, style: StrokeStyle(lineWidth: lineWidth, lineCap: .round))
                    .rotationEffect(.degrees(spin ? 270 : -90))
                    .animation(.linear(duration: 1.6).repeatForever(autoreverses: false), value: spin)
                    .onAppear { spin = true }
            }
            center()
        }
        .padding(lineWidth / 2)
        .frame(width: diameter, height: diameter)
    }
}
