import { ReactNode } from 'react'
import Sidebar from './Sidebar'
import styles from './Layout.module.css'

interface LayoutProps {
  children: ReactNode
}

function Layout({ children }: LayoutProps) {
  return (
    <div className={styles.layout}>
      <Sidebar />
      <main className={styles.content}>
        <div className={styles.contentInner}>
          {children}
        </div>
      </main>
    </div>
  )
}

export default Layout
