# Frontend Styling Guidelines

## Tailwind CSS v4

The project uses **Tailwind CSS v4** with enhanced features and improved performance.

### Key Features
- **Native CSS support**: Better integration with modern CSS features
- **Improved performance**: Faster build times and smaller bundle sizes
- **Enhanced dark mode**: Better dark mode utilities and system preference detection
- **New color system**: Improved color palette with better contrast ratios

## Dark Mode Implementation

### System Preference Detection
- Dark mode is enabled by default with automatic system preference detection
- Users can manually toggle between light and dark modes
- Theme preference is persisted in localStorage

### Dark Mode Classes
Use Tailwind's dark mode utilities:
```css
/* Light mode */
bg-white text-gray-900

/* Dark mode */
dark:bg-gray-900 dark:text-white
```

### Color Scheme Guidelines
- **Primary colors**: Use consistent primary colors that work in both modes
- **Background colors**: 
  - Light: `bg-white`, `bg-gray-50`, `bg-gray-100`
  - Dark: `bg-gray-900`, `bg-gray-800`, `bg-gray-700`
- **Text colors**:
  - Light: `text-gray-900`, `text-gray-700`, `text-gray-500`
  - Dark: `dark:text-white`, `dark:text-gray-200`, `dark:text-gray-400`
- **Border colors**:
  - Light: `border-gray-200`, `border-gray-300`
  - Dark: `dark:border-gray-700`, `dark:border-gray-600`

## Component Styling Standards

### Casting UI Components
All casting-related UI components must:
- Support both light and dark modes
- Use consistent spacing and typography
- Follow mobile-first responsive design
- Include proper focus states for accessibility
- Use semantic color classes (e.g., `text-primary`, `bg-secondary`)

### Button Styles
```css
/* Primary button */
bg-primary text-white hover:bg-primary-dark 
dark:bg-primary-light dark:hover:bg-primary

/* Secondary button */
bg-gray-200 text-gray-900 hover:bg-gray-300
dark:bg-gray-700 dark:text-white dark:hover:bg-gray-600

/* Glass effect (existing pattern) */
glass p-4 rounded-xl hover:shadow-retro active:scale-95 transition-all
```

### Modal and Overlay Styles
```css
/* Modal backdrop */
bg-black bg-opacity-50 dark:bg-opacity-70

/* Modal content */
bg-white dark:bg-gray-800 rounded-lg shadow-xl
```

## Responsive Design

### Breakpoints
Follow Tailwind's standard breakpoints:
- `sm`: 640px and up
- `md`: 768px and up  
- `lg`: 1024px and up
- `xl`: 1280px and up

### Mobile-First Approach
- Design for mobile first, then enhance for larger screens
- Use responsive utilities: `text-sm md:text-base lg:text-lg`
- Ensure touch targets are at least 44px on mobile

## Animation and Transitions

### Consistent Transitions
Use consistent transition classes:
```css
transition-colors duration-200
transition-all duration-300
hover:scale-105 active:scale-95
```

### Dark Mode Transitions
Ensure smooth transitions when switching themes:
```css
transition-colors duration-200 ease-in-out
```

## Accessibility

### Focus States
Always include focus states for interactive elements:
```css
focus:outline-none focus:ring-2 focus:ring-primary focus:ring-offset-2
dark:focus:ring-offset-gray-800
```

### Color Contrast
- Ensure sufficient contrast ratios in both light and dark modes
- Test with accessibility tools
- Use semantic color names rather than specific color values

## Best Practices

### Class Organization
Order classes logically:
1. Layout (flex, grid, position)
2. Spacing (margin, padding)
3. Sizing (width, height)
4. Typography (font, text)
5. Colors (background, text, border)
6. Effects (shadow, opacity)
7. Responsive modifiers
8. Dark mode modifiers
9. State modifiers (hover, focus, active)

### Example:
```css
flex items-center justify-between p-4 w-full 
bg-white text-gray-900 border border-gray-200 rounded-lg shadow-sm
hover:bg-gray-50 focus:ring-2 focus:ring-primary
dark:bg-gray-800 dark:text-white dark:border-gray-700 dark:hover:bg-gray-700
```

### Component Consistency
- Reuse existing component patterns from the codebase
- Follow the established glass morphism design language
- Maintain consistency with existing button styles and spacing
- Use the existing color palette and design tokens

## Testing Dark Mode

### Manual Testing
- Test all components in both light and dark modes
- Verify theme switching works correctly
- Check that system preference detection works
- Ensure no color contrast issues

### Automated Testing
- Include dark mode in visual regression tests
- Test theme persistence across page reloads
- Verify accessibility compliance in both modes