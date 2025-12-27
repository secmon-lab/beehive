// Theme colors based on the Beehive logo (honeycomb and bee)
export const theme = {
  colors: {
    // Primary colors from logo
    primary: '#FDB714',      // Golden yellow/orange (main bee color)
    primaryDark: '#E5A200',  // Darker shade of yellow
    primaryLight: '#FFD54F', // Lighter shade of yellow

    // Secondary colors
    secondary: '#FF9800',    // Orange (honeycomb gradient)
    secondaryDark: '#E68900',
    secondaryLight: '#FFB84D',

    // Dark/neutral colors
    dark: '#2C2C2C',         // Dark gray/black (logo outline)
    darkLight: '#424242',
    text: '#212121',
    textLight: '#757575',

    // Background colors
    bg: '#FAFAFA',
    bgLight: '#FFFFFF',
    bgDark: '#F5F5F5',

    // Sidebar
    sidebarBg: '#2C2C2C',
    sidebarText: '#FFFFFF',
    sidebarHover: '#424242',
    sidebarActive: '#FDB714',

    // Status colors
    success: '#4CAF50',
    warning: '#FF9800',
    error: '#F44336',
    info: '#2196F3',

    // Border colors
    border: '#E0E0E0',
    borderLight: '#EEEEEE',
  },

  spacing: {
    xs: '4px',
    sm: '8px',
    md: '16px',
    lg: '24px',
    xl: '32px',
    xxl: '48px',
  },

  borderRadius: {
    sm: '4px',
    md: '8px',
    lg: '12px',
    xl: '16px',
  },

  fontSize: {
    xs: '12px',
    sm: '14px',
    md: '16px',
    lg: '18px',
    xl: '24px',
    xxl: '32px',
  },

  fontWeight: {
    normal: 400,
    medium: 500,
    semibold: 600,
    bold: 700,
  },

  shadow: {
    sm: '0 1px 3px rgba(0,0,0,0.12), 0 1px 2px rgba(0,0,0,0.24)',
    md: '0 3px 6px rgba(0,0,0,0.16), 0 3px 6px rgba(0,0,0,0.23)',
    lg: '0 10px 20px rgba(0,0,0,0.19), 0 6px 6px rgba(0,0,0,0.23)',
  },
}

export type Theme = typeof theme
