## SwiftUI styling best practices

- **Consistent Methodology**: Use SwiftUI native modifiers for styling (not CSS) across the entire iOS app
- **Avoid Fighting SwiftUI**: Work with SwiftUI's declarative patterns rather than fighting against them with workarounds
- **Maintain Design System**: Establish design tokens (Color, Font, spacing values) in a centralized file for consistency
- **Leverage Native Components**: Use SwiftUI built-in components and modifiers to reduce custom view code
- **Performance Considerations**: Keep view hierarchies shallow and use LazyVStack/LazyHStack for long lists
