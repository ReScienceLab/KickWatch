import Foundation

@Observable
final class AlertsViewModel {
    var alerts: [AlertDTO] = []
    var isLoading = false
    var error: String?

    func load(deviceID: String) async {
        isLoading = true
        error = nil
        do {
            alerts = try await APIClient.shared.fetchAlerts(deviceID: deviceID)
        } catch {
            self.error = error.localizedDescription
        }
        isLoading = false
    }

    func createAlert(deviceID: String, keyword: String, categoryID: String?, minPercent: Double) async {
        let req = CreateAlertRequest(
            device_id: deviceID,
            keyword: keyword,
            category_id: categoryID,
            min_percent: minPercent > 0 ? minPercent : nil
        )
        do {
            let alert = try await APIClient.shared.createAlert(req)
            alerts.insert(alert, at: 0)
        } catch {
            self.error = error.localizedDescription
        }
    }

    func toggleAlert(_ alert: AlertDTO) async {
        let req = UpdateAlertRequest(is_enabled: !alert.is_enabled, keyword: nil, category_id: nil, min_percent: nil)
        do {
            let updated = try await APIClient.shared.updateAlert(id: alert.id, req: req)
            if let idx = alerts.firstIndex(where: { $0.id == alert.id }) {
                alerts[idx] = updated
            }
        } catch {
            self.error = error.localizedDescription
        }
    }

    func deleteAlert(_ alert: AlertDTO) async {
        do {
            try await APIClient.shared.deleteAlert(id: alert.id)
            alerts.removeAll { $0.id == alert.id }
        } catch {
            self.error = error.localizedDescription
        }
    }
}
