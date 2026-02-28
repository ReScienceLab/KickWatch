## iOS adaptive design best practices

- **iPhone-First Development**: Design primarily for iPhone sizes (6.1", 6.7") as the main target
- **Size Classes**: Use SwiftUI environment size classes for iPad adaptation if needed in future
- **Fluid Layouts**: Use GeometryReader sparingly and prefer flexible HStack/VStack with spacing
- **Dynamic Type**: Support Dynamic Type for accessibility - use .font(.body), .font(.title), etc.
- **Test Across Devices**: Test on iPhone SE (small), iPhone 15 Pro (standard), iPhone 15 Pro Max (large) simulators
- **Touch-Friendly Design**: Ensure tap targets are at least 44x44pt (Apple HIG minimum) for all interactive elements
- **Performance on Device**: Optimize image loading and scrolling performance for real devices, not just simulator
- **Readable Typography**: Use system fonts and Dynamic Type to maintain readability across device sizes
- **Content Priority**: Show the most important information first with clear visual hierarchy
