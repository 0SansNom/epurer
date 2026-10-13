// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "Epurer",
    platforms: [
        .macOS(.v14)
    ],
    products: [
        .executable(name: "Epurer", targets: ["Epurer"])
    ],
    targets: [
        .executableTarget(
            name: "Epurer",
            path: "Sources/Epurer"
        ),
        .testTarget(
            name: "EpurerTests",
            dependencies: ["Epurer"],
            path: "Tests/EpurerTests"
        )
    ]
)
