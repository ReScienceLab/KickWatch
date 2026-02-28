import XCTest
import SwiftData
@testable import KickWatch

final class CampaignModelTests: XCTestCase {
    private var container: ModelContainer!
    private var context: ModelContext!

    override func setUp() {
        super.setUp()
        container = try! ModelContainer(
            for: Campaign.self,
            configurations: ModelConfiguration(isStoredInMemoryOnly: true)
        )
        context = ModelContext(container)
    }

    override func tearDown() {
        context = nil
        container = nil
        super.tearDown()
    }

    private func make(pid: String = "1", state: String = "live", deadline: Date = .distantFuture) -> Campaign {
        let c = Campaign(pid: pid, name: "Test", deadline: deadline, state: state)
        context.insert(c)
        return c
    }

    // MARK: - stateLabel

    func testStateLabelSuccessful() {
        XCTAssertEqual(make(state: "successful").stateLabel, "Funded")
    }

    func testStateLabelFailed() {
        XCTAssertEqual(make(state: "failed").stateLabel, "Failed")
    }

    func testStateLabelCanceled() {
        XCTAssertEqual(make(state: "canceled").stateLabel, "Canceled")
    }

    func testStateLabelLive() {
        XCTAssertEqual(make(state: "live").stateLabel, "Live")
    }

    func testStateLabelUnknownDefaultsToLive() {
        XCTAssertEqual(make(state: "anything").stateLabel, "Live")
    }

    // MARK: - daysLeft

    func testDaysLeftFutureDate() {
        let future = Calendar.current.date(byAdding: .day, value: 10, to: .now)!
        let c = make(deadline: future)
        XCTAssertGreaterThanOrEqual(c.daysLeft, 9)
        XCTAssertLessThanOrEqual(c.daysLeft, 10)
    }

    func testDaysLeftPastDateReturnsZero() {
        let past = Calendar.current.date(byAdding: .day, value: -5, to: .now)!
        XCTAssertEqual(make(deadline: past).daysLeft, 0)
    }

    func testDaysLeftTodayReturnsZero() {
        XCTAssertEqual(make(deadline: .now).daysLeft, 0)
    }

    func testDaysLeftDistantFutureIsPositive() {
        XCTAssertGreaterThan(make(deadline: .distantFuture).daysLeft, 0)
    }

    // MARK: - Default init values

    func testDefaultValues() {
        let c = Campaign(pid: "x", name: "Test")
        context.insert(c)
        XCTAssertEqual(c.blurb, "")
        XCTAssertEqual(c.photoURL, "")
        XCTAssertEqual(c.goalAmount, 0)
        XCTAssertEqual(c.goalCurrency, "USD")
        XCTAssertEqual(c.pledgedAmount, 0)
        XCTAssertEqual(c.state, "live")
        XCTAssertFalse(c.isWatched)
        XCTAssertEqual(c.categoryID, "")
        XCTAssertEqual(c.creatorName, "")
        XCTAssertEqual(c.percentFunded, 0)
    }
}

final class WatchlistAlertModelTests: XCTestCase {
    private var container: ModelContainer!
    private var context: ModelContext!

    override func setUp() {
        super.setUp()
        container = try! ModelContainer(
            for: WatchlistAlert.self,
            configurations: ModelConfiguration(isStoredInMemoryOnly: true)
        )
        context = ModelContext(container)
    }

    override func tearDown() {
        context = nil
        container = nil
        super.tearDown()
    }

    func testDefaultValues() {
        let alert = WatchlistAlert(keyword: "robots")
        context.insert(alert)
        XCTAssertFalse(alert.id.isEmpty)
        XCTAssertEqual(alert.keyword, "robots")
        XCTAssertNil(alert.categoryID)
        XCTAssertEqual(alert.minPercentFunded, 0)
        XCTAssertTrue(alert.isEnabled)
        XCTAssertNil(alert.lastMatchedAt)
    }

    func testCustomValues() {
        let alert = WatchlistAlert(
            id: "fixed-id",
            keyword: "games",
            categoryID: "16",
            minPercentFunded: 50,
            isEnabled: false
        )
        context.insert(alert)
        XCTAssertEqual(alert.id, "fixed-id")
        XCTAssertEqual(alert.categoryID, "16")
        XCTAssertEqual(alert.minPercentFunded, 50)
        XCTAssertFalse(alert.isEnabled)
    }
}

final class RecentSearchModelTests: XCTestCase {
    private var container: ModelContainer!
    private var context: ModelContext!

    override func setUp() {
        super.setUp()
        container = try! ModelContainer(
            for: RecentSearch.self,
            configurations: ModelConfiguration(isStoredInMemoryOnly: true)
        )
        context = ModelContext(container)
    }

    override func tearDown() {
        context = nil
        container = nil
        super.tearDown()
    }

    func testInitSetsQuery() {
        let rs = RecentSearch(query: "board games")
        context.insert(rs)
        XCTAssertEqual(rs.query, "board games")
    }

    func testSearchedAtDefaultsToNow() {
        let before = Date()
        let rs = RecentSearch(query: "q")
        context.insert(rs)
        let after = Date()
        XCTAssertGreaterThanOrEqual(rs.searchedAt, before)
        XCTAssertLessThanOrEqual(rs.searchedAt, after)
    }

    func testCustomSearchedAt() {
        let date = Date(timeIntervalSince1970: 0)
        let rs = RecentSearch(query: "q", searchedAt: date)
        context.insert(rs)
        XCTAssertEqual(rs.searchedAt, date)
    }
}
