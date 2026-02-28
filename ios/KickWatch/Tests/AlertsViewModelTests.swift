import XCTest
@testable import KickWatch

final class AlertsViewModelTests: XCTestCase {
    private var mock: MockAPIClient!
    private var vm: AlertsViewModel!

    override func setUp() {
        super.setUp()
        mock = MockAPIClient()
        vm = AlertsViewModel(client: mock)
    }

    // MARK: - load()

    func testLoadSetsAlerts() async {
        mock.alertsResponse = [
            MockAPIClient.makeAlertDTO(id: "a1"),
            MockAPIClient.makeAlertDTO(id: "a2")
        ]

        await vm.load(deviceID: "device-1")

        XCTAssertEqual(vm.alerts.count, 2)
        XCTAssertFalse(vm.isLoading)
        XCTAssertNil(vm.error)
    }

    func testLoadSetsErrorOnFailure() async {
        mock.shouldThrow = APIError.serverError(statusCode: 500)

        await vm.load(deviceID: "device-1")

        XCTAssertNotNil(vm.error)
        XCTAssertFalse(vm.isLoading)
        XCTAssertTrue(vm.alerts.isEmpty)
    }

    func testLoadClearsErrorBeforeRequest() async {
        vm.error = "stale"
        mock.alertsResponse = []

        await vm.load(deviceID: "device-1")

        XCTAssertNil(vm.error)
    }

    // MARK: - createAlert()

    func testCreateAlertInsertsAtIndex0() async {
        let existing = MockAPIClient.makeAlertDTO(id: "existing")
        vm.alerts = [existing]
        mock.createAlertResult = MockAPIClient.makeAlertDTO(id: "new")

        await vm.createAlert(deviceID: "d1", keyword: "robots")

        XCTAssertEqual(vm.alerts.count, 2)
        XCTAssertEqual(vm.alerts[0].id, "new")
        XCTAssertEqual(vm.alerts[1].id, "existing")
    }

    func testCreateAlertEmptyKeywordBecomesNil() async {
        mock.createAlertResult = MockAPIClient.makeAlertDTO()

        await vm.createAlert(deviceID: "d1", keyword: "")

        XCTAssertNil(mock.createAlertRequests.last?.keyword)
    }

    func testCreateAlertNonEmptyKeywordIsPreserved() async {
        mock.createAlertResult = MockAPIClient.makeAlertDTO()

        await vm.createAlert(deviceID: "d1", keyword: "board games")

        XCTAssertEqual(mock.createAlertRequests.last?.keyword, "board games")
    }

    func testCreateAlertZeroMinPercentBecomesNil() async {
        mock.createAlertResult = MockAPIClient.makeAlertDTO()

        await vm.createAlert(deviceID: "d1", minPercent: 0)

        XCTAssertNil(mock.createAlertRequests.last?.min_percent)
    }

    func testCreateAlertPositiveMinPercentIsPreserved() async {
        mock.createAlertResult = MockAPIClient.makeAlertDTO()

        await vm.createAlert(deviceID: "d1", minPercent: 50)

        XCTAssertEqual(mock.createAlertRequests.last?.min_percent, 50)
    }

    func testCreateAlertZeroVelocityThreshBecomesNil() async {
        mock.createAlertResult = MockAPIClient.makeAlertDTO()

        await vm.createAlert(deviceID: "d1", velocityThresh: 0)

        XCTAssertNil(mock.createAlertRequests.last?.velocity_thresh)
    }

    func testCreateAlertSetsErrorOnFailure() async {
        mock.shouldThrow = APIError.serverError(statusCode: 422)

        await vm.createAlert(deviceID: "d1", keyword: "test")

        XCTAssertNotNil(vm.error)
        XCTAssertTrue(vm.alerts.isEmpty)
    }

    // MARK: - toggleAlert()

    func testToggleAlertFlipsIsEnabled() async {
        let alert = MockAPIClient.makeAlertDTO(id: "a1", isEnabled: true)
        vm.alerts = [alert]
        mock.updateAlertResult = MockAPIClient.makeAlertDTO(id: "a1", isEnabled: false)

        await vm.toggleAlert(alert)

        XCTAssertEqual(vm.alerts.first?.is_enabled, false)
    }

    func testToggleAlertSendsCorrectIDAndInvertedEnabled() async {
        let alert = MockAPIClient.makeAlertDTO(id: "a1", isEnabled: false)
        vm.alerts = [alert]
        mock.updateAlertResult = MockAPIClient.makeAlertDTO(id: "a1", isEnabled: true)

        await vm.toggleAlert(alert)

        let updateReq = mock.updateAlertRequests.last
        XCTAssertEqual(updateReq?.id, "a1")
        XCTAssertEqual(updateReq?.req.is_enabled, true)
    }

    func testToggleAlertUpdatesInPlaceByID() async {
        let a1 = MockAPIClient.makeAlertDTO(id: "a1", isEnabled: true)
        let a2 = MockAPIClient.makeAlertDTO(id: "a2", isEnabled: true)
        vm.alerts = [a1, a2]
        mock.updateAlertResult = MockAPIClient.makeAlertDTO(id: "a1", isEnabled: false)

        await vm.toggleAlert(a1)

        XCTAssertEqual(vm.alerts.count, 2)
        XCTAssertEqual(vm.alerts[0].is_enabled, false)
        XCTAssertEqual(vm.alerts[1].is_enabled, true)
    }

    func testToggleAlertSetsErrorOnFailure() async {
        let alert = MockAPIClient.makeAlertDTO(id: "a1")
        vm.alerts = [alert]
        mock.shouldThrow = APIError.serverError(statusCode: 500)

        await vm.toggleAlert(alert)

        XCTAssertNotNil(vm.error)
        XCTAssertEqual(vm.alerts.first?.is_enabled, alert.is_enabled)
    }

    // MARK: - deleteAlert()

    func testDeleteAlertRemovesFromList() async {
        let a1 = MockAPIClient.makeAlertDTO(id: "a1")
        let a2 = MockAPIClient.makeAlertDTO(id: "a2")
        vm.alerts = [a1, a2]

        await vm.deleteAlert(a1)

        XCTAssertEqual(vm.alerts.count, 1)
        XCTAssertEqual(vm.alerts.first?.id, "a2")
    }

    func testDeleteAlertCallsAPIWithCorrectID() async {
        let alert = MockAPIClient.makeAlertDTO(id: "del-99")
        vm.alerts = [alert]

        await vm.deleteAlert(alert)

        XCTAssertEqual(mock.deleteAlertIDs.last, "del-99")
    }

    func testDeleteAlertSetsErrorOnFailure() async {
        let alert = MockAPIClient.makeAlertDTO(id: "a1")
        vm.alerts = [alert]
        mock.shouldThrow = APIError.serverError(statusCode: 500)

        await vm.deleteAlert(alert)

        XCTAssertNotNil(vm.error)
        XCTAssertEqual(vm.alerts.count, 1)
    }
}
