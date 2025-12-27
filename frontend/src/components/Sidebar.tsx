import { Link, useLocation } from 'react-router-dom'
import styles from './Sidebar.module.css'

function Sidebar() {
  const location = useLocation()

  const isActive = (path: string) => {
    if (path === '/' && location.pathname === '/') return true
    if (path !== '/' && location.pathname.startsWith(path)) return true
    return false
  }

  return (
    <aside className={styles.sidebar}>
      <Link to="/" className={styles.logoLink}>
        <div className={styles.logo}>
          <img src="/logo.png" alt="Beehive" className={styles.logoImage} />
          <span className={styles.logoText}>Beehive</span>
        </div>
      </Link>

      <nav className={styles.nav}>
        <ul className={styles.navList}>
          <li className={styles.navItem}>
            <Link
              to="/ioc"
              className={`${styles.navLink} ${isActive('/ioc') ? styles.active : ''}`}
            >
              <span className={styles.icon}>🎯</span>
              <span>IoCs</span>
            </Link>
          </li>
          <li className={styles.navItem}>
            <Link
              to="/sources"
              className={`${styles.navLink} ${isActive('/sources') ? styles.active : ''}`}
            >
              <span className={styles.icon}>📡</span>
              <span>Sources</span>
            </Link>
          </li>
        </ul>
      </nav>
    </aside>
  )
}

export default Sidebar
