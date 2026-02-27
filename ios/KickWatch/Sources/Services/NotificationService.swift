import Foundation
import UserNotifications

@MainActor
final class NotificationService: ObservableObject {
    static let shared = NotificationService()
    private let deviceIDKey = "kickwatch.deviceID"

    @Published var isAuthorized = false

    func requestPermission() async {
        let center = UNUserNotificationCenter.current()
        let granted = (try? await center.requestAuthorization(options: [.alert, .badge, .sound])) ?? false
        isAuthorized = granted
    }

    func checkAuthorizationStatus() async {
        let settings = await UNUserNotificationCenter.current().notificationSettings()
        isAuthorized = settings.authorizationStatus == .authorized
    }

    func registerDeviceToken(_ tokenData: Data) async {
        let token = tokenData.map { String(format: "%02x", $0) }.joined()
        do {
            let response = try await APIClient.shared.registerDevice(token: token)
            KeychainHelper.save(response.device_id, for: deviceIDKey)
        } catch {
            print("NotificationService: failed to register device token: \(error)")
        }
    }

    var deviceID: String? {
        KeychainHelper.load(for: deviceIDKey)
    }
}
