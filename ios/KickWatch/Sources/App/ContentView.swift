import SwiftUI

struct ContentView: View {
    var body: some View {
        TabView {
            DiscoverView()
                .tabItem { Label("Discover", systemImage: "safari") }

            WatchlistView()
                .tabItem { Label("Watchlist", systemImage: "heart.fill") }

            AlertsView()
                .tabItem { Label("Alerts", systemImage: "bell.fill") }

            SettingsView()
                .tabItem { Label("Settings", systemImage: "gearshape") }
        }
    }
}
