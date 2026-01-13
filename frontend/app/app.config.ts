export default defineAppConfig({
  ui: {
    colors: {
      primary: 'attic',
      secondary: 'terracotta',
      neutral: 'mist'
    },
    button: {
      defaultVariants: {
        color: 'primary'
      }
    },
    card: {
      slots: {
        root: 'bg-white dark:bg-mist-800 ring-1 ring-mist-200 dark:ring-mist-700 rounded-xl shadow-card'
      }
    },
    input: {
      slots: {
        root: 'rounded-lg'
      }
    }
  }
})
