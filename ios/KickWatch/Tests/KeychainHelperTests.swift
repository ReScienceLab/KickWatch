import XCTest
@testable import KickWatch

final class KeychainHelperTests: XCTestCase {
    private let key = "com.test.kickwatch.keychain.unit"

    override func tearDown() {
        KeychainHelper.delete(for: key)
        super.tearDown()
    }

    func testSaveAndLoad() {
        KeychainHelper.save("hello", for: key)
        XCTAssertEqual(KeychainHelper.load(for: key), "hello")
    }

    func testSaveOverwritesExistingValue() {
        KeychainHelper.save("first", for: key)
        KeychainHelper.save("second", for: key)
        XCTAssertEqual(KeychainHelper.load(for: key), "second")
    }

    func testDeleteRemovesValue() {
        KeychainHelper.save("value", for: key)
        KeychainHelper.delete(for: key)
        XCTAssertNil(KeychainHelper.load(for: key))
    }

    func testLoadMissingKeyReturnsNil() {
        XCTAssertNil(KeychainHelper.load(for: key + ".nonexistent"))
    }

    func testDeleteNonexistentKeyDoesNotCrash() {
        KeychainHelper.delete(for: key + ".nonexistent")
    }

    func testSaveEmptyString() {
        KeychainHelper.save("", for: key)
        XCTAssertEqual(KeychainHelper.load(for: key), "")
    }

    func testSaveUnicodeValue() {
        KeychainHelper.save("日本語🎮", for: key)
        XCTAssertEqual(KeychainHelper.load(for: key), "日本語🎮")
    }

    func testSaveLongValue() {
        let longValue = String(repeating: "a", count: 4096)
        KeychainHelper.save(longValue, for: key)
        XCTAssertEqual(KeychainHelper.load(for: key), longValue)
    }
}
