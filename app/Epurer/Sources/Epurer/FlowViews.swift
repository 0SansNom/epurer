import SwiftUI

/// Hosts the main flow under "Accueil".
struct FlowView: View {
    @Bindable var model: AppModel

    var body: some View {
        Group {
            switch model.stage {
            case .idle: HomeScreen(model: model)
            case .scanning: ScanScreen(model: model)
            case .report: ReportScreen(model: model)
            case .review: ReviewScreen(model: model)
            case .confirm: ConfirmScreen(model: model)
            case .cleaning: CleanScreen(model: model)
            case .result: ResultScreen(model: model)
            }
        }
        .transition(.opacity)
        .animation(.easeInOut(duration: 0.2), value: model.stage)
    }
}

// MARK: 1 · Accueil

struct HomeScreen: View {
    @Bindable var model: AppModel
    @AppStorage(LastClean.bytesKey) private var lastBytes = 0
    @AppStorage(LastClean.dateKey) private var lastDate = 0.0

    private static let ago: RelativeDateTimeFormatter = {
        let f = RelativeDateTimeFormatter()
        f.unitsStyle = .full
        return f
    }()

    private var mood: Mascot.Mood {
        if model.detectState == .loading && model.domains.isEmpty { return .neutre }
        return model.homeStacks().shown.isEmpty ? .endormi : .content
    }

    var body: some View {
        ZStack(alignment: .bottom) {
            hero
            StackFooter(model: model)
                .padding(.bottom, 30)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var hero: some View {
        VStack(spacing: 0) {
            ProgressRing(fraction: model.disk?.fraction ?? 0) {
                VStack(spacing: 0) {
                    Mascot(mood: mood, size: 112)
                    if let disk = model.disk {
                        BigNumber(text: "\(Int((disk.fraction * 100).rounded())) %", size: 52, weight: .light)
                            .padding(.top, 2)
                        Text("Disque utilisé · \(Fmt.bytes(disk.used)) sur \(Fmt.bytes(disk.total))")
                            .foregroundStyle(Palette.secondary)
                            .padding(.top, 6)
                    }
                }
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(model.disk.map { String(localized: "Disque utilisé à \(Int(($0.fraction * 100).rounded())) %") } ?? String(localized: "Disque"))

            Button("Scanner") { model.scan() }
                .buttonStyle(ScanOrbStyle())
                .keyboardShortcut(.defaultAction)
                .padding(.top, 40)

            HStack(spacing: 12) {
                Button("Nettoyage sûr ›") { model.destination = .safeClean }
                    .buttonStyle(LinkStyle())
                    .fontWeight(.semibold)
                if lastDate > 0 {
                    Text("·").foregroundStyle(Palette.faint)
                    Text("Dernier nettoyage · \(Fmt.bytes(Int64(lastBytes))) · \(Self.ago.localizedString(for: Date(timeIntervalSince1970: lastDate), relativeTo: Date()))")
                        .foregroundStyle(Palette.secondary)
                }
            }
            .frame(height: 36)
            .padding(.top, 30)

            if let notice = model.notice {
                Text(notice)
                    .font(.system(size: 12))
                    .foregroundStyle(Palette.muted)
                    .padding(.top, 8)
            }
        }
        .padding(.bottom, 64)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

/// Detected stacks under the hero (1, 1a, 1b, 1d). Fixed height so the
/// loading bars give way to the stacks without a jump.
struct StackFooter: View {
    @Bindable var model: AppModel
    @State private var open: String?

    var body: some View {
        let (shown, extra) = model.homeStacks()
        VStack(spacing: 8) {
            if model.detectState == .loading && model.domains.isEmpty {
                HStack(spacing: 22) {
                    ForEach([64, 56, 52, 36], id: \.self) { width in
                        Capsule().fill(Palette.accent.opacity(0.12)).frame(width: CGFloat(width), height: 10)
                    }
                }
                .frame(height: 18)
                caption(String(localized: "Détection des stacks…"))
            } else if shown.isEmpty {
                Text("Aucune stack détectée.\nSeuls les caches système seront analysés.")
                    .font(.system(size: 12))
                    .foregroundStyle(Palette.secondary)
                    .multilineTextAlignment(.center)
                    .lineSpacing(3)
            } else {
                HStack(spacing: 22) {
                    ForEach(shown) { stack in
                        Button { open = open == stack.id ? nil : stack.id } label: {
                            HStack(spacing: 7) {
                                Dot(color: Palette.domain(stack.domain), size: 8)
                                Text(EngineTarget.domainLabel(stack.domain))
                            }
                            .contentShape(Rectangle())
                        }
                        .buttonStyle(.plain)
                        .overlay(alignment: .bottom) {
                            if open == stack.id {
                                bubble(stack).fixedSize().offset(y: -34)
                            }
                        }
                        .accessibilityHint("Afficher le détail")
                    }
                    if extra > 0 {
                        Text("+\(extra)")
                            .help(model.homeStacks(limit: .max).shown.dropFirst(shown.count).map { EngineTarget.domainLabel($0.domain) }.joined(separator: ", "))
                    }
                }
                .font(.system(size: 13))
                .foregroundStyle(Palette.secondary)
                .frame(height: 18)
                .opacity(model.detectState == .loading ? 0.5 : 1)
            }
        }
    }

    private func caption(_ text: String) -> some View {
        Text(text)
            .font(.system(size: 11))
            .foregroundStyle(Palette.muted)
    }

    /// 1d · One-line detail, the only place to re-run detection.
    private func bubble(_ stack: DomainStatus) -> some View {
        HStack(spacing: 10) {
            Dot(color: Palette.domain(stack.domain), size: 8)
            Text(EngineTarget.domainLabel(stack.domain))
                .font(.system(size: 13, weight: .semibold))
                .foregroundStyle(Palette.ink)
            Text((stack.tools.isEmpty ? stack.cleaners : stack.tools).joined(separator: ", "))
                .font(.system(size: 12))
                .foregroundStyle(Palette.secondary)
            Button("Actualiser") { model.detect() }
                .buttonStyle(LinkStyle(size: 12))
                .fontWeight(.semibold)
                .disabled(model.detectState == .loading)
        }
        .padding(.leading, 16)
        .padding(.trailing, 8)
        .frame(height: 40)
        .background(Capsule().fill(Palette.surface))
        .overlay(Capsule().strokeBorder(Palette.ink.opacity(0.08)))
        .shadow(color: Color(hex: 0x2B3752, opacity: 0.10), radius: 14, y: 12)
    }
}

/// 1c · Only on a broken local build: the engine binary is not in the app.
struct MissingEngine: View {
    var disk: DiskUsage?
    @State private var copied = false

    private let command = "make app-bundle"

    var body: some View {
        ZStack(alignment: .bottom) {
            VStack(spacing: 0) {
                Mascot(mood: .gene, size: 112)
                Text("Le moteur d’analyse est introuvable")
                    .font(.system(size: 26, weight: .light))
                    .tracking(-0.6)
                    .padding(.top, 20)
                Text("Cette version d’Épurer est incomplète. Reconstruisez l’app pour la réparer.")
                    .font(.system(size: 14))
                    .foregroundStyle(Palette.secondary)
                    .multilineTextAlignment(.center)
                    .lineSpacing(4)
                    .frame(width: 420)
                    .padding(.top, 8)
                HStack(spacing: 10) {
                    Text(command).font(.system(size: 13, design: .monospaced))
                    Button(copied ? "Copié" : "Copier") {
                        NSPasteboard.general.clearContents()
                        NSPasteboard.general.setString(command, forType: .string)
                        copied = true
                    }
                    .buttonStyle(LinkStyle())
                    .fontWeight(.semibold)
                    .padding(.horizontal, 6)
                }
                .padding(.leading, 18)
                .padding(.trailing, 8)
                .frame(height: 44)
                .background(Capsule().fill(Palette.accent.opacity(0.06)))
                .overlay(Capsule().strokeBorder(Palette.line))
                .padding(.top, 22)

                Button("Scanner") {}
                    .buttonStyle(ScanOrbStyle())
                    .disabled(true)
                    .padding(.top, 30)
            }
            .padding(.bottom, 64)
            .frame(maxWidth: .infinity, maxHeight: .infinity)

            if let disk {
                Text("Disque utilisé · \(Fmt.bytes(disk.used)) sur \(Fmt.bytes(disk.total)) · \(Int((disk.fraction * 100).rounded())) %")
                    .font(.system(size: 12))
                    .foregroundStyle(Palette.secondary)
                    .padding(.bottom, 30)
            }
        }
    }
}

// MARK: 2 · Scan en cours

struct ScanScreen: View {
    @Bindable var model: AppModel

    var body: some View {
        RingProgress(
            fraction: nil,
            mood: .neutre,
            tally: model.scanTally,
            // Grouped number ("1 842") shown; the trailing count only chooses the plural form.
            caption: String(localized: "\(Fmt.count(model.scanTally.items)) éléments analysés \(model.scanTally.items)"),
            label: String(localized: "Scan en cours"),
            buttonTitle: String(localized: "Annuler"),
            action: model.cancel
        )
    }
}

/// Big gradient ring with Pépin, a running size and the current path:
/// scan (2), clean (6) and safe-clean detection (8a).
struct RingProgress: View {
    var fraction: Double?
    var mood: Mascot.Mood
    var tally: ProgressTally
    var caption: String
    var label: String
    var buttonTitle: String
    var action: () -> Void

    var body: some View {
        VStack(spacing: 0) {
            ProgressRing(fraction: fraction) {
                VStack(spacing: 0) {
                    Mascot(mood: mood, size: 80)
                    BigNumber(text: Fmt.bytes(tally.bytes), size: 56, weight: .light)
                        .padding(.top, 4)
                        .padding(.bottom, 10)
                    Text(caption)
                        .foregroundStyle(Palette.muted)
                }
                .padding(.horizontal, 30)
            }
            .accessibilityElement(children: .combine)
            .accessibilityLabel(label)

            MonoPath(text: displayPath(tally.path))
                .padding(.top, 30)

            Button(buttonTitle, action: action)
                .buttonStyle(QuietStyle())
                .keyboardShortcut(.cancelAction)
                .padding(.top, 30)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

// MARK: 3 · Estimation

struct ReportScreen: View {
    @Bindable var model: AppModel

    private var approx: String { model.accessLimited ? "≥ " : "" }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack {
                if model.accessLimited {
                    Button { model.showPermission = true } label: {
                        HStack(spacing: 8) {
                            Dot(color: Palette.moderate, size: 6)
                            Text("Tailles minimales")
                        }
                        .font(.system(size: 12))
                        .foregroundStyle(Palette.muted)
                    }
                    .buttonStyle(.plain)
                    .help("Sans accès complet au disque, certaines tailles sont sous-estimées.")
                }
                Spacer()
                LevelTabs(level: model.level) { model.scan(level: $0) }
            }
            .frame(height: 32)

            if model.targets.isEmpty {
                NothingFoundCard(model: model)
                    .frame(maxWidth: .infinity, maxHeight: .infinity)
            } else {
                BigNumber(text: approx + Fmt.bytes(model.totalBytes), size: 112)
                    .padding(.top, 30)
                Text("Espace récupérable · \(String(localized: "\(model.targets.count) éléments")) · \(model.level.scope)")
                    .font(.system(size: 14))
                    .foregroundStyle(Palette.muted)
                    .padding(.top, 10)

                ScrollView {
                    VStack(spacing: 0) {
                        let maxBytes = model.domainTotals.first?.bytes ?? 1
                        ForEach(model.domainTotals) { row in
                            DomainBar(
                                name: EngineTarget.domainLabel(row.domain),
                                color: Palette.domain(row.domain),
                                fraction: Double(row.bytes) / Double(max(maxBytes, 1)),
                                size: approx + Fmt.bytes(row.bytes)
                            )
                        }
                    }
                }
                .scrollIndicators(.never)
                .padding(.top, 40)

                HStack {
                    if !model.locked.isEmpty {
                        Text("\(String(localized: "\(model.locked.count) éléments verrouillés")) (≥ \(Fmt.bytes(model.lockedBytes))), exclus du total")
                            .font(.system(size: 12))
                            .foregroundStyle(Palette.muted)
                            .help("Hors total : cet utilisateur ne peut pas les supprimer.")
                    }
                    Spacer()
                    Button("Examiner") { model.stage = .review }
                        .buttonStyle(PrimaryPillStyle())
                        .keyboardShortcut(.defaultAction)
                }
                .padding(.top, 16)
            }
        }
        .padding(EdgeInsets(top: 32, leading: 72, bottom: 40, trailing: 72))
        .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .topLeading)
    }
}

struct LevelTabs: View {
    var level: ScanLevel
    var onSelect: (ScanLevel) -> Void

    var body: some View {
        HStack(spacing: 22) {
            ForEach(ScanLevel.allCases) { item in
                let active = item == level
                Button(item.label) { if !active { onSelect(item) } }
                    .buttonStyle(.plain)
                    .font(.system(size: 13, weight: active ? .semibold : .regular))
                    .foregroundStyle(active ? Palette.ink : Palette.muted)
                    .frame(height: 32)
                    .overlay(alignment: .bottom) {
                        Rectangle().fill(active ? Palette.accent : .clear).frame(height: 2)
                    }
                    .accessibilityAddTraits(active ? .isSelected : [])
            }
        }
    }
}

struct DomainBar: View {
    var name: String
    var color: Color
    var fraction: Double
    var size: String

    var body: some View {
        HStack(spacing: 20) {
            Text(name)
                .font(.system(size: 15))
                .frame(width: 120, alignment: .leading)
            GeometryReader { geo in
                ZStack(alignment: .leading) {
                    Rectangle().fill(Palette.line).frame(height: 2)
                    Capsule().fill(color).frame(width: max(4, geo.size.width * fraction), height: 4)
                }
                .frame(maxHeight: .infinity)
            }
            .frame(height: 4)
            Text(size)
                .font(.system(size: 15))
                .monospacedDigit()
                .frame(width: 90, alignment: .trailing)
        }
        .frame(height: 54)
        .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
    }
}

/// "Rien trouvé" (Pépin étonné) with a way to widen the level.
struct NothingFoundCard: View {
    @Bindable var model: AppModel

    var body: some View {
        StateCard(ring: .dashed, title: String(localized: "Rien trouvé"), text: String(localized: "Aucun cache à ce niveau.")) {
            Mascot(mood: .surpris, size: 84)
        } action: {
            if let next = model.nextLevel {
                Button("Passer en \(next.label)") { model.scan(level: next) }
                    .buttonStyle(LinkStyle(color: Palette.accent))
            } else {
                Button("Rescanner") { model.scan() }
                    .buttonStyle(LinkStyle(color: Palette.accent))
            }
        }
    }
}

// MARK: 4 · Review & sélection

struct ReviewScreen: View {
    @Bindable var model: AppModel

    var body: some View {
        VStack(spacing: 0) {
            HStack(alignment: .firstTextBaseline) {
                Button("‹ Estimation") { model.stage = .report }
                    .buttonStyle(LinkStyle(size: 14))
                    .padding(.trailing, 18)
                Text("Sélection")
                    .font(.system(size: 30, weight: .light))
                    .tracking(-1)
                Spacer()
                HStack(spacing: 10) {
                    Button("Tout") { model.selectAll(true) }
                        .help("Cocher tout sauf les éléments risqués")
                    Text("·").foregroundStyle(Palette.muted)
                    Button("Aucun") { model.selectAll(false) }
                }
                .buttonStyle(LinkStyle())
                .padding(.trailing, 18)
                Text("\(String(localized: "\(model.chosen.count) sélectionnés")) · \(Fmt.bytes(model.chosenBytes))")
                    .font(.system(size: 14))
                    .foregroundStyle(Palette.muted)
                    .monospacedDigit()
            }
            .padding(.top, 36)
            .padding(.bottom, 18)

            ScrollView {
                LazyVStack(spacing: 0) {
                    ForEach(model.targets) { target in
                        TargetRow(target: target, checked: model.selected.contains(target.id)) {
                            model.toggle(target)
                        }
                    }
                }
            }
            .scrollIndicators(.automatic)

            HStack(spacing: 28) {
                if model.chosen.isEmpty {
                    VStack(alignment: .leading, spacing: 2) {
                        Text("0 élément sélectionné").font(.system(size: 14))
                        Text("Cochez au moins une ligne.")
                            .font(.system(size: 12))
                            .foregroundStyle(Palette.muted)
                    }
                    Spacer()
                    Button {} label: {
                        Text("Nettoyer 0")
                            .font(.system(size: 13, weight: .semibold))
                            .foregroundStyle(Palette.secondary)
                            .padding(.horizontal, 22)
                            .frame(height: 34)
                            .background(Capsule().fill(Palette.accent.opacity(0.06)))
                            .overlay(Capsule().strokeBorder(Palette.ink.opacity(0.12)))
                    }
                    .buttonStyle(.plain)
                    .disabled(true)
                } else {
                    HStack(spacing: 20) {
                        DotLabel(color: Palette.safe, text: Safety.safe.label)
                        DotLabel(color: Palette.moderate, text: Safety.moderate.label)
                        DotLabel(color: Palette.danger, text: Safety.dangerous.label)
                    }
                    .font(.system(size: 12))
                    .foregroundStyle(Palette.muted)
                    Spacer()
                    Button("Prévisualiser") { model.previewSelection { model.stage = .confirm } }
                        .buttonStyle(LinkStyle())
                        .fontWeight(.semibold)
                    Button("Nettoyer \(Fmt.count(model.chosen.count))") {
                        model.stage = .confirm
                    }
                    .buttonStyle(PrimaryPillStyle(height: 48, horizontalPadding: 36))
                    .keyboardShortcut(.defaultAction)
                }
            }
            .frame(height: 84)
            .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
        }
        .padding(.horizontal, 72)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

struct TargetRow: View {
    var target: EngineTarget
    var checked: Bool
    var onToggle: () -> Void

    var body: some View {
        Button(action: onToggle) {
            HStack(spacing: 16) {
                ZStack {
                    if checked {
                        Circle().fill(Palette.accent)
                        Image(systemName: "checkmark")
                            .font(.system(size: 9, weight: .heavy))
                            .foregroundStyle(.white)
                    } else {
                        Circle().strokeBorder(Palette.muted, lineWidth: 1.4)
                    }
                }
                .frame(width: 18, height: 18)

                VStack(alignment: .leading, spacing: 1) {
                    Text(target.label)
                        .font(.system(size: 15))
                        .lineLimit(1)
                    HStack(spacing: 0) {
                        Text("\(target.domainLabel) · ")
                        Text(target.displayPath).font(.system(size: 11, design: .monospaced))
                    }
                    .font(.system(size: 11))
                    .foregroundStyle(Palette.muted)
                    .lineLimit(1)
                    .truncationMode(.middle)
                }
                .frame(maxWidth: .infinity, alignment: .leading)

                Text(Fmt.bytes(target.sizeBytes))
                    .font(.system(size: 15))
                    .monospacedDigit()
                    .frame(width: 80, alignment: .trailing)

                DotLabel(color: Palette.safety(target.safetyLevel), text: target.safetyLevel.label)
                    .font(.system(size: 12))
                    .foregroundStyle(Palette.muted)
                    .frame(width: 84, alignment: .leading)
            }
            .frame(height: 56)
            .contentShape(Rectangle())
            .opacity(checked ? 1 : 0.5)
            .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
        }
        .buttonStyle(.plain)
        .accessibilityLabel("\(target.label), \(Fmt.bytes(target.sizeBytes)), \(target.safetyLevel.label)")
        .accessibilityValue(checked ? "Sélectionné" : "Ignoré")
        .contextMenu {
            Button("Afficher dans le Finder") {
                NSWorkspace.shared.activateFileViewerSelecting([URL(fileURLWithPath: target.path)])
            }
        }
    }
}

// MARK: 5 · Confirmation

struct ConfirmScreen: View {
    @Bindable var model: AppModel

    private static let columnRows = 5

    var body: some View {
        let chosen = model.chosen
        let groups = [Safety.safe, .moderate, .dangerous].compactMap { level -> (Safety, [EngineTarget])? in
            let items = chosen.filter { $0.safetyLevel == level }
            return items.isEmpty ? nil : (level, items)
        }

        VStack(spacing: 0) {
            Text("Nettoyer \(chosen.count) éléments ?")
                .font(.system(size: 15))
                .foregroundStyle(Palette.muted)
            BigNumber(text: Fmt.bytes(model.chosenBytes), size: 84)
                .padding(.top, 8)
            Text(model.includesTrash(chosen) ? "seront libérés · suppression définitive, Corbeille comprise" : "seront libérés · suppression définitive")
                .foregroundStyle(Palette.muted)
                .padding(.top, 8)

            HStack(alignment: .top, spacing: 40) {
                ForEach(groups, id: \.0) { level, items in
                    ConfirmColumn(level: level, items: items, maxRows: Self.columnRows)
                }
            }
            .padding(.top, 30)

            Button("Nettoyer") { model.clean() }
                .buttonStyle(HaloPillStyle())
                .keyboardShortcut(.defaultAction)
                .padding(.top, 30)

            Button("Annuler") { model.stage = .review }
                .buttonStyle(QuietStyle())
                .keyboardShortcut(.cancelAction)
                .padding(.top, 10)
        }
        .padding(.horizontal, 40)
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

struct ConfirmColumn: View {
    var level: Safety
    var items: [EngineTarget]
    var maxRows: Int

    private var title: String {
        switch level {
        case .safe: return String(localized: "Sûrs")
        case .moderate: return String(localized: "Modérés")
        case .dangerous: return String(localized: "Risqués")
        }
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 0) {
            HStack(spacing: 8) {
                Dot(color: Palette.safety(level))
                Text("\(title) · \(Fmt.count(items.count))")
            }
            .font(.system(size: 13, weight: .semibold))
            .frame(height: 30)

            ForEach(items.prefix(maxRows)) { item in
                HStack {
                    Text(item.label).lineLimit(1).truncationMode(.middle)
                    Spacer(minLength: 12)
                    Text(Fmt.bytes(item.sizeBytes))
                        .foregroundStyle(Palette.muted)
                        .monospacedDigit()
                }
                .font(.system(size: 13))
                .frame(height: 38)
                .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
            }
            if items.count > maxRows {
                let rest = items.dropFirst(maxRows)
                HStack {
                    Text("+ \(rest.count) autres")
                    Spacer()
                    Text(Fmt.bytes(rest.reduce(0) { $0 + $1.sizeBytes })).monospacedDigit()
                }
                .font(.system(size: 13))
                .foregroundStyle(Palette.muted)
                .frame(height: 38)
                .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
            }
        }
        .frame(width: 290)
    }
}

// MARK: 6 · Nettoyage en cours

struct CleanScreen: View {
    @Bindable var model: AppModel

    var body: some View {
        let total = model.cleanedTargets.count
        let done = min(model.cleanTally.items, total)
        RingProgress(
            fraction: model.cleanFraction,
            mood: .concentre,
            tally: model.cleanTally,
            // The two trailing counts only choose singular or plural; the catalog never prints them.
            caption: String(localized: "libérés · \(Fmt.count(done)) éléments sur \(Fmt.count(total)) \(done) \(total)"),
            label: String(localized: "Nettoyage à \(Int(model.cleanFraction * 100)) %"),
            buttonTitle: String(localized: "Arrêter"),
            action: model.cancel
        )
    }
}

// MARK: 7 · Résultat

struct ResultScreen: View {
    @Bindable var model: AppModel

    var body: some View {
        let ok = model.results.filter(\.success).count
        let failed = model.results.filter { $0.status == "failed" }.count
        let left = model.results.filter { $0.status == "left_in_place" }.count
        let issues = model.results.filter { !$0.success }
        VStack(spacing: 0) {
            Mascot(mood: issues.isEmpty ? .content : .gene, size: 84)
            Text("Libérés")
                .font(.system(size: 14))
                .foregroundStyle(Palette.muted)
                .padding(.top, 6)
            BigNumber(text: Fmt.bytes(model.freedBytes), size: 120)
                .padding(.top, 6)

            HStack(spacing: 22) {
                DotLabel(color: Palette.safe, text: String(localized: "\(ok) nettoyés"))
                if failed > 0 {
                    DotLabel(color: Palette.danger, text: String(localized: "\(failed) échecs"))
                }
                if left > 0 {
                    DotLabel(color: Palette.moderate, text: String(localized: "\(left) laissés en place"))
                }
            }
            .foregroundStyle(Palette.muted)
            .padding(.top, 14)

            if !issues.isEmpty {
                ScrollView {
                    VStack(spacing: 0) {
                        ForEach(issues) { issue in
                            IssueRow(result: issue, target: model.target(at: issue.path))
                        }
                    }
                }
                .frame(width: 480)
                .frame(maxHeight: 54 * 3)
                .padding(.top, 30)
            }

            Button("Terminé") { model.finish() }
                .buttonStyle(PrimaryPillStyle(height: 50, horizontalPadding: 44))
                .keyboardShortcut(.defaultAction)
                .padding(.top, 26)
            if failed > 0 {
                // Apple's rule for text that does not show the number: one string per case, not a plural variant.
                Button(failed == 1 ? String(localized: "Réessayer l’échec") : String(localized: "Réessayer les échecs")) {
                    model.retryFailed()
                }
                .buttonStyle(LinkStyle())
                .fontWeight(.semibold)
                .padding(.top, 14)
            }
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

struct IssueRow: View {
    var result: CleanResult
    var target: EngineTarget?

    private var note: String {
        let reason = result.error.map { " · \($0)" } ?? ""
        return (result.status == "left_in_place" ? String(localized: "Laissé en place") : String(localized: "Échec")) + reason
    }

    var body: some View {
        HStack(spacing: 14) {
            VStack(alignment: .leading, spacing: 1) {
                Text(target?.label ?? result.path)
                    .font(.system(size: 14))
                    .lineLimit(1)
                Text(note)
                    .font(.system(size: 11))
                    .foregroundStyle(Palette.muted)
                    .lineLimit(1)
                    .truncationMode(.tail)
            }
            .frame(maxWidth: .infinity, alignment: .leading)
            if let target {
                Text(Fmt.bytes(target.sizeBytes)).font(.system(size: 14)).monospacedDigit()
            }
        }
        .frame(height: 54)
        .overlay(alignment: .top) { Rectangle().fill(Palette.line).frame(height: 1) }
        .help(result.error ?? result.path)
    }
}

func displayPath(_ path: String) -> String {
    let home = FileManager.default.homeDirectoryForCurrentUser.path
    if path.hasPrefix(home) {
        return "~" + path.dropFirst(home.count)
    }
    return path.isEmpty ? " " : path
}
