import SwiftUI
import Charts

struct SparklineView: View {
    let pid: String

    @State private var snapshots: [CampaignSnapshotDTO] = []
    @State private var isLoading = true

    private static let dateFormatter: ISO8601DateFormatter = {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withFullDate]
        return f
    }()

    var body: some View {
        Group {
            if snapshots.count >= 2 {
                chart
            } else if isLoading {
                Color.clear
            } else {
                Color.clear
            }
        }
        .frame(width: 64, height: 28)
        .task(id: pid) {
            isLoading = true
            snapshots = (try? await APIClient.shared.fetchCampaignHistory(pid: pid)) ?? []
            isLoading = false
        }
    }

    private var chart: some View {
        Chart(indexedSnapshots, id: \.index) { item in
            LineMark(
                x: .value("Day", item.index),
                y: .value("Pledged", item.snapshot.pledged_amount)
            )
            .foregroundStyle(lineColor)
            .lineStyle(StrokeStyle(lineWidth: 1.5))

            AreaMark(
                x: .value("Day", item.index),
                yStart: .value("Base", minValue),
                yEnd: .value("Pledged", item.snapshot.pledged_amount)
            )
            .foregroundStyle(lineColor.opacity(0.15))
        }
        .chartXAxis(.hidden)
        .chartYAxis(.hidden)
        .chartXScale(domain: 0...(indexedSnapshots.count - 1))
        .chartYScale(domain: minValue...maxValue)
    }

    private var indexedSnapshots: [(index: Int, snapshot: CampaignSnapshotDTO)] {
        snapshots.enumerated().map { (index: $0.offset, snapshot: $0.element) }
    }

    private var minValue: Double {
        (snapshots.map(\.pledged_amount).min() ?? 0) * 0.95
    }

    private var maxValue: Double {
        let max = snapshots.map(\.pledged_amount).max() ?? 1
        return max * 1.05
    }

    private var lineColor: Color {
        guard snapshots.count >= 2 else { return .accentColor }
        let first = snapshots.first!.pledged_amount
        let last = snapshots.last!.pledged_amount
        return last >= first ? Color.green : Color.orange
    }
}
