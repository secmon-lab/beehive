import { useQuery } from '@apollo/client'
import { useNavigate } from 'react-router-dom'
import { LIST_SOURCES } from '../graphql/queries'
import { formatRelativeTime } from '../utils/time'
import styles from './SourceList.module.css'

interface SourceState {
  sourceID: string
  lastFetchedAt?: string
  lastItemID?: string
  lastItemDate?: string
  itemCount: number
  errorCount: number
  lastStatus?: string
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
  const navigate = useNavigate()
  const { loading, error, data } = useQuery<ListSourcesData>(LIST_SOURCES)

  const handleRowClick = (sourceId: string, event: React.MouseEvent) => {
    // Don't navigate if clicking on a link
    if ((event.target as HTMLElement).tagName === 'A') {
      return
    }
    navigate(`/sources/${sourceId}`)
  }

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
                <th>Status</th>
              </tr>
            </thead>
            <tbody>
              {sources.map((source) => (
                <tr
                  key={source.id}
                  onClick={(e) => handleRowClick(source.id, e)}
                  className={styles.clickableRow}
                >
                  <td>{source.id}</td>
                  <td>
                    <span
                      className={`${styles.typeChip} ${
                        source.type === 'rss' ? styles.typeRss : styles.typeFeed
                      }`}
                    >
                      {source.type}
                    </span>
                  </td>
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
                  <td
                    title={
                      source.state?.lastFetchedAt
                        ? new Date(source.state.lastFetchedAt).toLocaleString()
                        : undefined
                    }
                  >
                    {source.state?.lastFetchedAt
                      ? formatRelativeTime(source.state.lastFetchedAt)
                      : '-'}
                  </td>
                  <td>
                    {source.state?.lastStatus ? (
                      <span
                        className={`${styles.badge} ${
                          source.state.lastStatus === 'success'
                            ? styles.badgeSuccess
                            : source.state.lastStatus === 'error'
                            ? styles.badgeError
                            : styles.badgePartial
                        }`}
                      >
                        {source.state.lastStatus}
                      </span>
                    ) : (
                      '-'
                    )}
                  </td>
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
