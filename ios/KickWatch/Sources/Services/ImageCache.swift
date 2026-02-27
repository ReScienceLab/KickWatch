import SwiftUI

actor ImageCache {
    static let shared = ImageCache()
    private var cache: [URL: Image] = [:]
    private let session: URLSession

    init(session: URLSession = .shared) {
        self.session = session
    }

    func image(for url: URL) async -> Image? {
        if let cached = cache[url] { return cached }
        guard let (data, _) = try? await session.data(from: url),
              let uiImage = UIImage(data: data) else { return nil }
        let image = Image(uiImage: uiImage)
        cache[url] = image
        return image
    }
}

struct RemoteImage: View {
    let urlString: String
    @State private var image: Image?

    var body: some View {
        Group {
            if let image {
                image.resizable().scaledToFill()
            } else {
                Rectangle().fill(Color(.systemGray5))
                    .task {
                        guard let url = URL(string: urlString) else { return }
                        image = await ImageCache.shared.image(for: url)
                    }
            }
        }
    }
}
