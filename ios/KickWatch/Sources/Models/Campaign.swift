import Foundation
import SwiftData

@Model
final class Campaign {
    @Attribute(.unique) var pid: String
    var name: String
    var blurb: String
    var photoURL: String
    var goalAmount: Double
    var goalCurrency: String
    var pledgedAmount: Double
    var deadline: Date
    var state: String
    var categoryName: String
    var categoryID: String
    var projectURL: String
    var creatorName: String
    var percentFunded: Double
    var isWatched: Bool
    var lastFetchedAt: Date

    init(
        pid: String,
        name: String,
        blurb: String = "",
        photoURL: String = "",
        goalAmount: Double = 0,
        goalCurrency: String = "USD",
        pledgedAmount: Double = 0,
        deadline: Date = .distantFuture,
        state: String = "live",
        categoryName: String = "",
        categoryID: String = "",
        projectURL: String = "",
        creatorName: String = "",
        percentFunded: Double = 0,
        isWatched: Bool = false,
        lastFetchedAt: Date = .now
    ) {
        self.pid = pid
        self.name = name
        self.blurb = blurb
        self.photoURL = photoURL
        self.goalAmount = goalAmount
        self.goalCurrency = goalCurrency
        self.pledgedAmount = pledgedAmount
        self.deadline = deadline
        self.state = state
        self.categoryName = categoryName
        self.categoryID = categoryID
        self.projectURL = projectURL
        self.creatorName = creatorName
        self.percentFunded = percentFunded
        self.isWatched = isWatched
        self.lastFetchedAt = lastFetchedAt
    }

    var daysLeft: Int {
        max(0, Calendar.current.dateComponents([.day], from: .now, to: deadline).day ?? 0)
    }

    var stateLabel: String {
        switch state {
        case "successful": return "Funded"
        case "failed": return "Failed"
        case "canceled": return "Canceled"
        default: return "Live"
        }
    }
}
