import { useQuery } from '@apollo/client'
import { LIST_SOURCES } from '../graphql/queries'
import styles from './SourceList.module.css'

interface SourceState {
  sourceID: string
  lastFetchedAt?: string
  lastItemID?: string
  lastItemDate?: string
  itemCount: number
  errorCount: number
  lastError?: string
  updatedAt: string
}

interface Source {
  id: string
  type: string
  url: string
  tags: string[]
  enabled: boolean
  state?: SourceState
}

interface ListSourcesData {
  listSources: Source[]
}

function SourceList() {
  const { loading, error, data } = useQuery<ListSourcesData>(LIST_SOURCES)

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading sources...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>Error loading sources: {error.message}</div>
      </div>
    )
  }

  const sources = data?.listSources || []

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.title}>Sources</h1>
        <p className={styles.subtitle}>Configured IoC sources</p>
      </div>

      {sources.length === 0 ? (
        <div className={styles.empty}>No sources found</div>
      ) : (
        <div className={styles.tableContainer}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Source ID</th>
                <th>Type</th>
                <th>URL</th>
                <th>Tags</th>
                <th>Enabled</th>
                <th>Last Fetched</th>
                <th>Item Count</th>
                <th>Error Count</th>
              </tr>
            </thead>
            <tbody>
              {sources.map((source) => (
                <tr key={source.id}>
                  <td>{source.id}</td>
                  <td>{source.type}</td>
                  <td>
                    <a
                      href={source.url}
                      target="_blank"
                      rel="noopener noreferrer"
                      className={styles.url}
                    >
                      {source.url}
                    </a>
                  </td>
                  <td>
                    {source.tags.length > 0 ? (
                      <div className={styles.tags}>
                        {source.tags.map((tag, idx) => (
                          <span key={idx} className={styles.tag}>
                            {tag}
                          </span>
                        ))}
                      </div>
                    ) : (
                      '-'
                    )}
                  </td>
                  <td>
                    <span
                      className={`${styles.badge} ${
                        source.enabled ? styles.badgeEnabled : styles.badgeDisabled
                      }`}
                    >
                      {source.enabled ? 'Enabled' : 'Disabled'}
                    </span>
                  </td>
                  <td>
                    {source.state?.lastFetchedAt
                      ? new Date(source.state.lastFetchedAt).toLocaleString()
                      : '-'}
                  </td>
                  <td>{source.state?.itemCount ?? '-'}</td>
                  <td>{source.state?.errorCount ?? '-'}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

export default SourceList
