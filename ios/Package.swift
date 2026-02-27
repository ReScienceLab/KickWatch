// swift-tools-version: 5.9
import PackageDescription

let package = Package(
    name: "KickWatch",
    platforms: [.iOS(.v17)],
    products: [
        .library(name: "KickWatch", targets: ["KickWatch"]),
    ],
    targets: [
        .target(
            name: "KickWatch",
            path: "KickWatch/Sources"
        ),
    ]
)
