function Dashboard() {
  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'center',
      justifyContent: 'center',
      minHeight: '100vh',
      fontFamily: 'system-ui, -apple-system, sans-serif',
      backgroundColor: '#f5f5f5',
    }}>
      <div style={{
        padding: '2rem',
        backgroundColor: 'white',
        borderRadius: '8px',
        boxShadow: '0 2px 8px rgba(0, 0, 0, 0.1)',
        textAlign: 'center',
      }}>
        <h1 style={{
          fontSize: '2.5rem',
          margin: '0 0 1rem 0',
          color: '#333',
        }}>
          🐝 Beehive
        </h1>
        <p style={{
          fontSize: '1.2rem',
          margin: '0',
          color: '#666',
        }}>
          IoC Management System
        </p>
      </div>
    </div>
  )
}

export default Dashboard
