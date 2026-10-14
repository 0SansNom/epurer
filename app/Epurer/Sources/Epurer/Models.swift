import Foundation

struct EngineInfo: Decodable, Sendable {
    let protocolVersion: Int
    let command: String
    let version: String
    let privilegedHelper: Bool
    let asksAdminPassword: Bool
    let diskAccess: String
    let hardenedUserSpace: Bool
    let supportedCommands: [String]

    enum CodingKeys: String, CodingKey {
        case protocolVersion = "protocol"
        case command, version
        case privilegedHelper = "privileged_helper"
        case asksAdminPassword = "asks_admin_password"
        case diskAccess = "disk_access"
        case hardenedUserSpace = "hardened_user_space"
        case supportedCommands = "supported_commands"
    }
}

struct EngineTarget: Codable, Identifiable, Hashable, Sendable {
    var id: String { path + "|" + cleaner }
    let path: String
    let description: String
    let sizeBytes: Int64
    let safety: String
    let domain: String
    let cleaner: String

    enum CodingKeys: String, CodingKey {
        case path, description, safety, domain, cleaner
        case sizeBytes = "size_bytes"
    }
}

struct ScanResponse: Decodable, Sendable {
    let protocolVersion: Int
    let command: String
    let diskAccess: String
    let targets: [EngineTarget]
    let locked: [EngineTarget]

    enum CodingKeys: String, CodingKey {
        case protocolVersion = "protocol"
        case command
        case diskAccess = "disk_access"
        case targets, locked
    }
}

struct CleanRequest: Encodable, Sendable {
    let protocolVersion: Int
    let dryRun: Bool
    let targets: [EngineTarget]

    enum CodingKeys: String, CodingKey {
        case protocolVersion = "protocol"
        case dryRun = "dry_run"
        case targets
    }
}

struct CleanResult: Decodable, Identifiable, Sendable {
    var id: String { path }
    let path: String
    let success: Bool
    let bytesFreed: Int64
    let error: String?
    let status: String

    enum CodingKeys: String, CodingKey {
        case path, success, error, status
        case bytesFreed = "bytes_freed"
    }
}

struct CleanResponse: Decodable, Sendable {
    let protocolVersion: Int
    let command: String
    let results: [CleanResult]

    enum CodingKeys: String, CodingKey {
        case protocolVersion = "protocol"
        case command, results
    }
}

struct ProgressEvent: Decodable, Sendable {
    let protocolVersion: Int
    let event: String
    let phase: String
    let path: String
    let items: Int
    let bytes: Int64
    var total: Int? = nil
    var detail: String? = nil

    enum CodingKeys: String, CodingKey {
        case protocolVersion = "protocol"
        case event, phase, path, items, bytes, total, detail
    }
}

// MARK: - Display helpers

enum Safety: String, Sendable {
    case safe = "Safe"
    case moderate = "Moderate"
    case dangerous = "Dangerous"

    init(engine: String) {
        self = Safety(rawValue: engine) ?? .safe
    }

    var label: String {
        switch self {
        case .safe: return String(localized: "Sûr")
        case .moderate: return String(localized: "Modéré")
        case .dangerous: return String(localized: "Risqué")
        }
    }
}

extension EngineTarget {
    var safetyLevel: Safety { Safety(engine: safety) }

    /// French name for the engine's domain string.
    var domainLabel: String { Self.domainLabel(domain) }

    static func domainLabel(_ domain: String) -> String {
        domain == "System" ? String(localized: "Système") : domain
    }

    /// The engine's English description, translated through the catalog.
    /// Paths the user picked (Explorer, Projets) keep their real file names.
    var label: String {
        cleaner == "paths" ? description : description.engineLocalized
    }

    /// Path with the home directory folded to `~`.
    var displayPath: String { Epurer.displayPath(path) }
}

// MARK: - Analyze / purge / ignore

struct DiskEntry: Decodable, Identifiable, Hashable, Sendable {
    var id: String { path }
    let name: String
    let path: String
    let sizeBytes: Int64
    let isDir: Bool
    let incomplete: Bool
    let removable: Bool
    let ignored: Bool

    enum CodingKeys: String, CodingKey {
        case name, path, incomplete, removable, ignored
        case sizeBytes = "size_bytes"
        case isDir = "is_dir"
    }
}

struct AnalyzeResponse: Decodable, Sendable {
    let path: String
    let diskAccess: String
    let entries: [DiskEntry]

    enum CodingKeys: String, CodingKey {
        case path, entries
        case diskAccess = "disk_access"
    }
}

struct ProjectArtifact: Decodable, Identifiable, Hashable, Sendable {
    var id: String { path }
    let path: String
    let kind: String
    let sizeBytes: Int64
    let safety: String
    let selected: Bool

    enum CodingKeys: String, CodingKey {
        case path, kind, safety, selected
        case sizeBytes = "size_bytes"
    }
}

struct DevProject: Decodable, Identifiable, Sendable {
    var id: String { root }
    let root: String
    let name: String
    let type: String
    let totalBytes: Int64
    let artifacts: [ProjectArtifact]

    enum CodingKeys: String, CodingKey {
        case root, name, type, artifacts
        case totalBytes = "total_bytes"
    }
}

struct PurgeResponse: Decodable, Sendable {
    let roots: [String]
    let projects: [DevProject]
}

struct DomainStatus: Decodable, Identifiable, Sendable {
    var id: String { domain }
    let domain: String
    let detected: Bool
    let tools: [String]
    let cleaners: [String]
}

struct DetectResponse: Decodable, Sendable {
    let domains: [DomainStatus]
}

struct IgnoreResponse: Decodable, Sendable {
    let paths: [String]
    /// "system" when the engine refused an add: the path is out of reach anyway.
    var skipped: String?
}

extension EngineTarget {
    /// A user-picked path, removed by the engine's "paths" cleaner.
    static func picked(path: String, description: String, sizeBytes: Int64, safety: String = "Moderate") -> EngineTarget {
        EngineTarget(path: path, description: description, sizeBytes: sizeBytes, safety: safety, domain: "System", cleaner: "paths")
    }
}

extension String {
    /// Engine names and descriptions are English (the CLI shows them as is);
    /// the catalog holds their French. "User caches: Google" translates the
    /// part before ": " and keeps the name. Unknown text comes back unchanged.
    var engineLocalized: String {
        if let colon = range(of: ": ") {
            let prefix = String(self[..<colon.lowerBound]).engineLocalized
            let name = String(self[colon.upperBound...])
            return String(localized: "\(prefix): \(name)", comment: "Engine group and item, e.g. User caches: Google")
        }
        return String(localized: String.LocalizationValue(self))
    }
}
