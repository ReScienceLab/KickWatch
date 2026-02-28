import SwiftUI
import Charts

struct SparklineView: View {
    let dataPoints: [HistoryDataPoint]
    let height: CGFloat = 60

    private var trendColor: Color {
        guard dataPoints.count >= 2 else { return .gray }
        let first = dataPoints.first?.pledged_amount ?? 0
        let last = dataPoints.last?.pledged_amount ?? 0
        if last > first * 1.05 { return .green }
        if last < first * 0.95 { return .red }
        return .gray
    }

    var body: some View {
        if dataPoints.isEmpty {
            emptyState
        } else {
            chart
        }
    }

    private var emptyState: some View {
        HStack {
            Image(systemName: "chart.line.uptrend.xyaxis")
                .font(.system(size: 24))
                .foregroundStyle(.secondary)
            Text("No trend data yet")
                .font(.caption).foregroundStyle(.secondary)
        }
        .frame(height: height)
        .frame(maxWidth: .infinity)
    }

    private var chart: some View {
        Chart(dataPoints) { point in
            AreaMark(
                x: .value("Date", point.date ?? Date()),
                y: .value("Pledged", point.pledged_amount)
            )
            .foregroundStyle(
                LinearGradient(
                    colors: [trendColor.opacity(0.3), trendColor.opacity(0.05)],
                    startPoint: .top,
                    endPoint: .bottom
                )
            )

            LineMark(
                x: .value("Date", point.date ?? Date()),
                y: .value("Pledged", point.pledged_amount)
            )
            .foregroundStyle(trendColor)
            .lineStyle(StrokeStyle(lineWidth: 2, lineCap: .round))
        }
        .chartXAxis(.hidden)
        .chartYAxis(.hidden)
        .frame(height: height)
    }
}
