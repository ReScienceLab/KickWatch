import SwiftUI
import SwiftData

@main
struct KickWatchApp: App {
    let container: ModelContainer
    @UIApplicationDelegateAdaptor(AppDelegate.self) var appDelegate

    private static let schemaVersion = 1

    init() {
        let defaults = UserDefaults.standard
        if defaults.integer(forKey: "schemaVersion") != Self.schemaVersion {
            Self.deleteStore()
            defaults.set(Self.schemaVersion, forKey: "schemaVersion")
        }
        do {
            container = try ModelContainer(for: Campaign.self, WatchlistAlert.self, RecentSearch.self)
        } catch {
            Self.deleteStore()
            container = try! ModelContainer(for: Campaign.self, WatchlistAlert.self, RecentSearch.self)
        }
    }

    private static func deleteStore() {
        let url = URL.applicationSupportDirectory.appending(path: "default.store")
        for ext in ["", "-wal", "-shm"] {
            try? FileManager.default.removeItem(at: url.appendingPathExtension(ext))
        }
    }

    var body: some Scene {
        WindowGroup {
            ContentView()
        }
        .modelContainer(container)
    }
}
