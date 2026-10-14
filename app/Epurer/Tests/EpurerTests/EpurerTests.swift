import XCTest
@testable import Epurer

final class ProgressTallyTests: XCTestCase {
    private func event(_ path: String, _ items: Int, _ bytes: Int64) -> ProgressEvent {
        ProgressEvent(protocolVersion: 1, event: "progress", phase: "scan", path: path, items: items, bytes: bytes)
    }

    func testAddsUpBatchesThatRestartFromZero() {
        var tally = ProgressTally()
        tally.add(event("/a/1", 1, 10))
        tally.add(event("/a/2", 2, 30))
        // A new walk or cleaner restarts its own counters.
        tally.add(event("/b/1", 1, 5))
        tally.add(event("/c/1", 1, 7))

        XCTAssertEqual(tally.items, 4)
        XCTAssertEqual(tally.bytes, 42)
        XCTAssertEqual(tally.path, "/c/1")
    }
}

final class FmtTests: XCTestCase {
    /// No string catalog in the package, so units are the French keys;
    /// the decimal separator follows the current locale.
    func testBytesUseDecimalUnits() {
        let sep = Locale.current.decimalSeparator ?? "."
        XCTAssertEqual(Fmt.bytes(0), "0 o")
        XCTAssertEqual(Fmt.bytes(999), "999 o")
        XCTAssertEqual(Fmt.bytes(12_400_000_000), "12\(sep)4 Go")
        XCTAssertEqual(Fmt.bytes(150_000_000), "150 Mo")
    }
}

@MainActor
final class ProtectedModelTests: XCTestCase {
    func testCoveredPaths() {
        let model = ProtectedModel()
        model.paths = ["/Users/me/Projects"]

        XCTAssertEqual(model.coveringPath("/Users/me/Projects/client"), "/Users/me/Projects")
        XCTAssertEqual(model.coveringPath("/Users/me/Projects"), "/Users/me/Projects")
        // A sibling that only shares a prefix is not covered.
        XCTAssertNil(model.coveringPath("/Users/me/Projects-old"))

        XCTAssertEqual(ProtectedModel.normalize("~/a/../b"), NSHomeDirectory() + "/b")
    }
}

final class ProtectedPlanTests: XCTestCase {
    func testBatchSplitsNewAndCoveredPaths() {
        let plan = ProtectedModel.plan(
            ["/p/Projects/a", "/p/X", "/p/X/y", "/p/X", "/p/Projects/b", "/p/Projects"],
            protected: ["/p/Projects"]
        )
        // Duplicates and already-protected paths are dropped; order is kept.
        XCTAssertEqual(plan.add, ["/p/X"])
        XCTAssertEqual(plan.covered.map(\.parent), ["/p/Projects", "/p/X", "/p/Projects"])
        XCTAssertEqual(plan.covered.map(\.child), ["a", "y", "b"])
    }
}

@MainActor
final class HomeTests: XCTestCase {
    private func stack(_ domain: String, _ detected: Bool = true) -> DomainStatus {
        DomainStatus(domain: domain, detected: detected, tools: [], cleaners: [])
    }

    func testStacksHideSystemAndAbsentAndCapAtFive() {
        let model = AppModel()
        model.domains = [stack("Frontend"), stack("System"), stack("DevOps", false),
                         stack("Backend"), stack("Mobile"), stack("Data/ML"), stack("AI"), stack("Go")]
        let (shown, extra) = model.homeStacks()
        XCTAssertEqual(shown.map(\.domain), ["Frontend", "Backend", "Mobile", "Data/ML", "AI"])
        XCTAssertEqual(extra, 1)
    }

    func testLastCleanCountsOnlyWhatWasFreed() {
        let ok = CleanResult(path: "/a", success: true, bytesFreed: 10, error: nil, status: "ok")
        let failed = CleanResult(path: "/b", success: false, bytesFreed: 0, error: "busy", status: "failed")
        XCTAssertEqual(LastClean.freed([ok, failed, ok]), 20)
        XCTAssertNil(LastClean.freed([failed]))
    }
}

@MainActor
final class ProjectsSearchTests: XCTestCase {
    private func event(_ kind: String, path: String = "", items: Int = 0, bytes: Int64 = 0, total: Int? = nil, detail: String? = nil) -> ProgressEvent {
        ProgressEvent(protocolVersion: 1, event: kind, phase: "purge", path: path, items: items, bytes: bytes, total: total, detail: detail)
    }

    func testFoundEventsGroupByProject() {
        let model = ProjectsModel()
        model.apply(event("roots", detail: "/p/Projects\n/p/Code"))
        model.apply(event("found", path: "/p/Projects/web/node_modules", items: 1, bytes: 30, detail: "Node.js"))
        model.apply(event("found", path: "/p/Code/api/target", items: 2, bytes: 5, detail: "Rust"))
        model.apply(event("found", path: "/p/Projects/web/.next", items: 3, bytes: 10, detail: "Node.js"))
        model.apply(event("progress", items: 4, total: 11))

        XCTAssertEqual(model.roots, ["/p/Projects", "/p/Code"])
        XCTAssertEqual(model.partial.map(\.name), ["web", "api"])
        XCTAssertEqual(model.partial[0].totalBytes, 40)
        XCTAssertEqual(model.partial[0].artifacts.map(\.kind), ["node_modules", ".next"])
        XCTAssertEqual(model.reading, "/p/Projects/web")
        XCTAssertEqual(model.searchDone, 4)
        XCTAssertEqual(model.searchTotal, 11)
    }
}

/// French is the catalog's source language but not the bundle's development
/// region (English), so a key without an explicit "fr" value is looked up in
/// English on a French Mac. Every key needs both.
final class LocalizationCatalogTests: XCTestCase {
    func testEveryKeyHasFrenchAndEnglish() throws {
        let url = URL(fileURLWithPath: #filePath)
            .deletingLastPathComponent().appendingPathComponent("../../../EpurerApp/Localizable.xcstrings")
        let catalog = try JSONSerialization.jsonObject(with: Data(contentsOf: url)) as! [String: Any]
        let strings = catalog["strings"] as! [String: [String: Any]]
        let missing = strings.filter { _, entry in
            guard entry["shouldTranslate"] as? Bool != false else { return false }
            let loc = entry["localizations"] as? [String: Any] ?? [:]
            return loc["fr"] == nil || loc["en"] == nil
        }.keys.sorted()
        XCTAssertEqual(missing, [], "Keys without a French or English value")
    }
}

/// The engine sends English names that the app translates at display time
/// (`String.engineLocalized`). A name added to the Go code without a catalog
/// entry would show in English on a French Mac.
final class EngineNamesCatalogTests: XCTestCase {
    func testEveryEngineNameIsInTheCatalog() throws {
        let root = URL(fileURLWithPath: #filePath).deletingLastPathComponent().appendingPathComponent("../../../..")
        let catalogURL = root.appendingPathComponent("app/EpurerApp/Localizable.xcstrings")
        let catalog = try JSONSerialization.jsonObject(with: Data(contentsOf: catalogURL)) as! [String: Any]
        let keys = Set((catalog["strings"] as! [String: Any]).keys)

        // The literal forms the cleaners use for names and descriptions.
        let patterns = [
            #"(?:Description|description):\s*"([^"]+)""#,
            #"desc\s*=\s*"([^"]+)""#,
            #"systemCleaner\{name:\s*"([^"]+)"\}"#,
        ]
        var names = Set<String>()
        let files = FileManager.default.enumerator(at: root.appendingPathComponent("internal"), includingPropertiesForKeys: nil)!
        for case let url as URL in files where url.pathExtension == "go" && !url.lastPathComponent.hasSuffix("_test.go") {
            let source = try String(contentsOf: url, encoding: .utf8)
            for pattern in patterns { names.formUnion(matches(pattern, in: source)) }
            // Values of the `descriptions := map[string]string{ … }` tables.
            for block in matches(#"descriptions := map\[string\]string\{([^}]*)\}"#, in: source) {
                names.formUnion(matches(#""[^"]*":\s*"([^"]+)""#, in: block))
            }
        }
        XCTAssertGreaterThan(names.count, 50, "The source scan found too few names; the patterns are out of date")
        XCTAssertEqual(names.subtracting(keys).sorted(), [], "Engine names missing from Localizable.xcstrings")
    }

    private func matches(_ pattern: String, in text: String) -> [String] {
        let regex = try! NSRegularExpression(pattern: pattern)
        return regex.matches(in: text, range: NSRange(text.startIndex..., in: text)).compactMap {
            Range($0.range(at: 1), in: text).map { String(text[$0]) }
        }
    }
}
