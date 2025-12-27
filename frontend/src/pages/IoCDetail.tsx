import { useQuery } from '@apollo/client'
import { Link, useParams } from 'react-router-dom'
import { GET_IOC } from '../graphql/queries'
import styles from './IoCDetail.module.css'

interface IoC {
  id: string
  sourceID: string
  sourceType: string
  type: string
  value: string
  description: string
  sourceURL?: string
  context: string
  status: string
  firstSeenAt: string
  updatedAt: string
}

interface GetIoCData {
  getIoC: IoC | null
}

function IoCDetail() {
  const { id } = useParams<{ id: string }>()
  const { loading, error, data } = useQuery<GetIoCData>(GET_IOC, {
    variables: { id },
  })

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading IoC details...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>Error loading IoC: {error.message}</div>
      </div>
    )
  }

  const ioc = data?.getIoC

  if (!ioc) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>IoC not found</div>
        <Link to="/ioc" className={styles.backLink}>
          ← Back to IoC List
        </Link>
      </div>
    )
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <Link to="/ioc" className={styles.backLink}>
          ← Back to IoC List
        </Link>
        <h1 className={styles.title}>{ioc.value}</h1>
        <p className={styles.subtitle}>IoC Details</p>
      </div>

      <div className={styles.card}>
        <h2 className={styles.sectionTitle}>Basic Information</h2>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>ID</div>
          <div className={styles.fieldValue}>{ioc.id}</div>
        </div>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>Type</div>
          <div className={styles.fieldValue}>{ioc.type}</div>
        </div>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>Value</div>
          <div className={styles.fieldValue}>{ioc.value}</div>
        </div>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>Status</div>
          <div className={styles.fieldValue}>
            <span
              className={`${styles.badge} ${
                ioc.status === 'active' ? styles.badgeActive : styles.badgeInactive
              }`}
            >
              {ioc.status}
            </span>
          </div>
        </div>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>Description</div>
          <div className={styles.fieldValue}>{ioc.description || '-'}</div>
        </div>
      </div>

      <div className={styles.card}>
        <h2 className={styles.sectionTitle}>Source Information</h2>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>Source ID</div>
          <div className={styles.fieldValue}>{ioc.sourceID}</div>
        </div>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>Source Type</div>
          <div className={styles.fieldValue}>{ioc.sourceType}</div>
        </div>

        {ioc.sourceURL && (
          <div className={styles.field}>
            <div className={styles.fieldLabel}>Source URL</div>
            <div className={styles.fieldValue}>
              <a href={ioc.sourceURL} target="_blank" rel="noopener noreferrer">
                {ioc.sourceURL}
              </a>
            </div>
          </div>
        )}
      </div>

      <div className={styles.card}>
        <h2 className={styles.sectionTitle}>Context</h2>
        <div className={styles.context}>{ioc.context || 'No context available'}</div>
      </div>

      <div className={styles.card}>
        <h2 className={styles.sectionTitle}>Timestamps</h2>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>First Seen</div>
          <div className={styles.fieldValue}>
            {new Date(ioc.firstSeenAt).toLocaleString()}
          </div>
        </div>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>Last Updated</div>
          <div className={styles.fieldValue}>
            {new Date(ioc.updatedAt).toLocaleString()}
          </div>
        </div>
      </div>
    </div>
  )
}

export default IoCDetail
