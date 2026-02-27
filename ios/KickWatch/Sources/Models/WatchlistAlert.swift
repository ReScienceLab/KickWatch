import Foundation
import SwiftData

@Model
final class WatchlistAlert {
    @Attribute(.unique) var id: String
    var keyword: String
    var categoryID: String?
    var minPercentFunded: Double
    var isEnabled: Bool
    var createdAt: Date
    var lastMatchedAt: Date?

    init(
        id: String = UUID().uuidString,
        keyword: String,
        categoryID: String? = nil,
        minPercentFunded: Double = 0,
        isEnabled: Bool = true,
        createdAt: Date = .now,
        lastMatchedAt: Date? = nil
    ) {
        self.id = id
        self.keyword = keyword
        self.categoryID = categoryID
        self.minPercentFunded = minPercentFunded
        self.isEnabled = isEnabled
        self.createdAt = createdAt
        self.lastMatchedAt = lastMatchedAt
    }
}
