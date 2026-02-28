import XCTest
@testable import KickWatch

final class DiscoverViewModelTests: XCTestCase {
    private var mock: MockAPIClient!
    private var vm: DiscoverViewModel!

    override func setUp() {
        super.setUp()
        mock = MockAPIClient()
        vm = DiscoverViewModel(client: mock)
    }

    // MARK: - load()

    func testLoadSetsCampaigns() async {
        mock.campaignListResponse = CampaignListResponse(
            campaigns: [MockAPIClient.makeCampaignDTO(pid: "p1")],
            next_cursor: nil, total: 1
        )

        await vm.load()

        XCTAssertEqual(vm.campaigns.count, 1)
        XCTAssertEqual(vm.campaigns.first?.pid, "p1")
    }

    func testLoadClearsIsLoadingOnSuccess() async {
        mock.campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: nil, total: 0)

        await vm.load()

        XCTAssertFalse(vm.isLoading)
        XCTAssertNil(vm.error)
    }

    func testLoadSetsHasMoreWhenCursorPresent() async {
        mock.campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: "c1", total: 0)

        await vm.load()

        XCTAssertTrue(vm.hasMore)
        XCTAssertEqual(vm.nextCursor, "c1")
    }

    func testLoadSetsHasMoreFalseWhenNoCursor() async {
        mock.campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: nil, total: 0)

        await vm.load()

        XCTAssertFalse(vm.hasMore)
        XCTAssertNil(vm.nextCursor)
    }

    func testLoadSetsErrorOnFailure() async {
        mock.shouldThrow = APIError.serverError(statusCode: 503)

        await vm.load()

        XCTAssertNotNil(vm.error)
        XCTAssertFalse(vm.isLoading)
        XCTAssertTrue(vm.campaigns.isEmpty)
    }

    func testLoadClearsErrorBeforeRequest() async {
        vm.error = "stale error"
        mock.campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: nil, total: 0)

        await vm.load()

        XCTAssertNil(vm.error)
    }

    func testLoadPassesSelectedSortAndCategory() async {
        vm.selectedSort = "magic"
        vm.selectedCategoryID = "16"
        mock.campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: nil, total: 0)

        await vm.load()

        let call = mock.fetchCampaignsCalls.last
        XCTAssertEqual(call?.sort, "magic")
        XCTAssertEqual(call?.categoryID, "16")
        XCTAssertNil(call?.cursor)
    }

    // MARK: - loadMore()

    func testLoadMoreDoesNothingWhenHasMoreFalse() async {
        vm.hasMore = false

        await vm.loadMore()

        XCTAssertTrue(mock.fetchCampaignsCalls.isEmpty)
    }

    func testLoadMoreDoesNothingWhenNoNextCursor() async {
        vm.hasMore = true
        vm.nextCursor = nil

        await vm.loadMore()

        XCTAssertTrue(mock.fetchCampaignsCalls.isEmpty)
    }

    func testLoadMoreAppendsCampaigns() async {
        // First page
        let p1 = MockAPIClient.makeCampaignDTO(pid: "p1")
        mock.campaignListResponse = CampaignListResponse(campaigns: [p1], next_cursor: "cur2", total: 2)
        await vm.load()

        // Second page
        let p2 = MockAPIClient.makeCampaignDTO(pid: "p2")
        mock.campaignListResponse = CampaignListResponse(campaigns: [p2], next_cursor: nil, total: 2)
        await vm.loadMore()

        XCTAssertEqual(vm.campaigns.count, 2)
        XCTAssertEqual(vm.campaigns[0].pid, "p1")
        XCTAssertEqual(vm.campaigns[1].pid, "p2")
        XCTAssertFalse(vm.hasMore)
    }

    func testLoadMorePassesCursorToAPI() async {
        mock.campaignListResponse = CampaignListResponse(
            campaigns: [MockAPIClient.makeCampaignDTO()], next_cursor: "cursor-xyz", total: 10
        )
        await vm.load()
        mock.campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: nil, total: 10)

        await vm.loadMore()

        XCTAssertEqual(mock.fetchCampaignsCalls.last?.cursor, "cursor-xyz")
    }

    func testLoadMoreSetsErrorOnFailure() async {
        mock.campaignListResponse = CampaignListResponse(
            campaigns: [MockAPIClient.makeCampaignDTO()], next_cursor: "c", total: 2
        )
        await vm.load()

        mock.shouldThrow = APIError.serverError(statusCode: 500)
        await vm.loadMore()

        XCTAssertNotNil(vm.error)
        XCTAssertFalse(vm.isLoadingMore)
    }

    // MARK: - loadCategories()

    func testLoadCategoriesFetchesAndStores() async {
        mock.categoriesResponse = [CategoryDTO(id: "1", name: "Art", parent_id: nil)]

        await vm.loadCategories()

        XCTAssertEqual(vm.categories.count, 1)
        XCTAssertEqual(vm.categories.first?.name, "Art")
    }

    func testLoadCategoriesSkipsIfAlreadyLoaded() async {
        vm.categories = [CategoryDTO(id: "1", name: "Art", parent_id: nil)]
        mock.fetchCategoriesCalled = false

        await vm.loadCategories()

        XCTAssertFalse(mock.fetchCategoriesCalled)
        XCTAssertEqual(vm.categories.count, 1)
    }

    // MARK: - selectSort()

    func testSelectSortUpdatesSelectedSort() async {
        mock.campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: nil, total: 0)

        await vm.selectSort("newest")

        XCTAssertEqual(vm.selectedSort, "newest")
    }

    func testSelectSortResetsCursorAndReloads() async {
        vm.nextCursor = "old-cursor"
        mock.campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: nil, total: 0)

        await vm.selectSort("magic")

        let call = mock.fetchCampaignsCalls.last
        XCTAssertNil(call?.cursor)
        XCTAssertEqual(call?.sort, "magic")
    }

    // MARK: - selectCategory()

    func testSelectCategoryUpdatesAndReloads() async {
        mock.campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: nil, total: 0)

        await vm.selectCategory("16")

        XCTAssertEqual(vm.selectedCategoryID, "16")
        XCTAssertEqual(mock.fetchCampaignsCalls.last?.categoryID, "16")
    }

    func testSelectCategoryNilClearsCategory() async {
        vm.selectedCategoryID = "16"
        mock.campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: nil, total: 0)

        await vm.selectCategory(nil)

        XCTAssertNil(vm.selectedCategoryID)
        XCTAssertNil(mock.fetchCampaignsCalls.last?.categoryID)
    }

    func testSelectCategoryResetsCursor() async {
        vm.nextCursor = "stale"
        mock.campaignListResponse = CampaignListResponse(campaigns: [], next_cursor: nil, total: 0)

        await vm.selectCategory("5")

        XCTAssertNil(mock.fetchCampaignsCalls.last?.cursor)
    }
}
