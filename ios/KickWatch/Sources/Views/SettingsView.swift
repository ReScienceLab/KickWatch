import SwiftUI

struct SettingsView: View {
    @StateObject private var notificationService = NotificationService.shared

    var body: some View {
        NavigationStack {
            Form {
                Section("Notifications") {
                    HStack {
                        Label("Push Notifications", systemImage: "bell")
                        Spacer()
                        if notificationService.isAuthorized {
                            Text("Enabled").foregroundStyle(.secondary)
                        } else {
                            Button("Enable") {
                                Task { await notificationService.requestPermission() }
                            }
                        }
                    }
                }
                Section("About") {
                    LabeledContent("Version", value: Bundle.main.infoDictionary?["CFBundleShortVersionString"] as? String ?? "—")
                    LabeledContent("Build", value: Bundle.main.infoDictionary?["CFBundleVersion"] as? String ?? "—")
                }
            }
            .navigationTitle("Settings")
            .task { await notificationService.checkAuthorizationStatus() }
        }
    }
}
