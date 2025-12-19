import styles from './Dashboard.module.css'

function Dashboard() {
  return (
    <div className={styles.container}>
      <div className={styles.card}>
        <h1 className={styles.title}>
          🐝 Beehive
        </h1>
        <p className={styles.subtitle}>
          IoC Management System
        </p>
      </div>
    </div>
  )
}

export default Dashboard
