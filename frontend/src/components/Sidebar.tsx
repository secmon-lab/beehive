import { Link, NavLink } from 'react-router-dom'
import styles from './Sidebar.module.css'

function Sidebar() {
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
            <NavLink
              to="/ioc"
              className={({ isActive }) =>
                `${styles.navLink} ${isActive ? styles.active : ''}`
              }
            >
              <span className={styles.icon}>🎯</span>
              <span>IoCs</span>
            </NavLink>
          </li>
          <li className={styles.navItem}>
            <NavLink
              to="/sources"
              className={({ isActive }) =>
                `${styles.navLink} ${isActive ? styles.active : ''}`
              }
            >
              <span className={styles.icon}>📡</span>
              <span>Sources</span>
            </NavLink>
          </li>
        </ul>
      </nav>
    </aside>
  )
}

export default Sidebar
