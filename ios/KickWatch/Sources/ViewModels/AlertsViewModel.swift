import Foundation

@Observable
final class AlertsViewModel {
    private let client: any APIClientProtocol

    var alerts: [AlertDTO] = []
    var isLoading = false
    var error: String?

    init(client: any APIClientProtocol = APIClient.shared) {
        self.client = client
    }

    func load(deviceID: String) async {
        isLoading = true
        error = nil
        do {
            alerts = try await client.fetchAlerts(deviceID: deviceID)
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    func createAlert(deviceID: String, alertType: String = "keyword", keyword: String = "", categoryID: String? = nil, minPercent: Double = 0, velocityThresh: Double = 0) async {
        let req = CreateAlertRequest(
            device_id: deviceID,
            alert_type: alertType,
            keyword: keyword.isEmpty ? nil : keyword,
            category_id: categoryID,
            min_percent: minPercent > 0 ? minPercent : nil,
            velocity_thresh: velocityThresh > 0 ? velocityThresh : nil
        )
        do {
            let alert = try await client.createAlert(req)
            alerts.insert(alert, at: 0)
        } catch {
            self.error = error.localizedDescription
        }
    }

    func toggleAlert(_ alert: AlertDTO) async {
        let req = UpdateAlertRequest(is_enabled: !alert.is_enabled, keyword: nil, category_id: nil, min_percent: nil)
        do {
            let updated = try await client.updateAlert(id: alert.id, req: req)
            if let idx = alerts.firstIndex(where: { $0.id == alert.id }) {
                alerts[idx] = updated
            }
        } catch {
            self.error = error.localizedDescription
        }
    }

    func deleteAlert(_ alert: AlertDTO) async {
        do {
            try await client.deleteAlert(id: alert.id)
            alerts.removeAll { $0.id == alert.id }
        } catch {
            self.error = error.localizedDescription
        }
    }
}
