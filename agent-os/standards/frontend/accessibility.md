## iOS accessibility best practices

- **Semantic Components**: Use appropriate SwiftUI components (Button, NavigationLink, Label) that convey meaning to VoiceOver
- **VoiceOver Support**: Ensure all interactive elements are accessible via VoiceOver with descriptive labels
- **Color Contrast**: Maintain sufficient contrast ratios and don't rely solely on color to convey information (test with accessibility inspector)
- **Accessibility Labels**: Provide descriptive `.accessibilityLabel()` for images and meaningful labels for all interactive elements
- **VoiceOver Testing**: Test and verify that all views are accessible with VoiceOver enabled on a physical device or simulator
- **Accessibility Hints**: Use `.accessibilityHint()` to provide additional context for complex interactions when needed
- **Logical Heading Structure**: Use proper view hierarchy and accessibility traits to create clear navigation structure
- **Focus Management**: Manage focus appropriately in sheets, alerts, and navigation transitions
