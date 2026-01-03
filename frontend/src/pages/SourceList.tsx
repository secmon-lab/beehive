import { useState } from 'react'
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
  schema?: string
  schemaDescription?: string
  description?: string
  tags: string[]
  enabled: boolean
  state?: SourceState
}

interface ListSourcesData {
  listSources: Source[]
}

type SortField = 'id' | 'type' | 'enabled' | 'lastFetchedAt' | 'lastStatus'
type SortOrder = 'asc' | 'desc'

function SourceList() {
  const navigate = useNavigate()
  const [sortField, setSortField] = useState<SortField>('id')
  const [sortOrder, setSortOrder] = useState<SortOrder>('asc')

  const { loading, error, data } = useQuery<ListSourcesData>(LIST_SOURCES)

  const handleRowClick = (sourceId: string, event: React.MouseEvent) => {
    // Don't navigate if clicking on a link
    if ((event.target as HTMLElement).tagName === 'A') {
      return
    }
    navigate(`/sources/${sourceId}`)
  }

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'asc' ? 'desc' : 'asc')
    } else {
      setSortField(field)
      setSortOrder('asc')
    }
  }

  const getSortIcon = (field: SortField) => {
    if (sortField !== field) return ' ↕'
    return sortOrder === 'asc' ? ' ↑' : ' ↓'
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

  // Client-side sorting
  const sortedSources = [...sources].sort((a, b) => {
    let aValue: string | number | boolean | undefined
    let bValue: string | number | boolean | undefined

    switch (sortField) {
      case 'id':
        aValue = a.id
        bValue = b.id
        break
      case 'type':
        aValue = a.type
        bValue = b.type
        break
      case 'enabled':
        aValue = a.enabled
        bValue = b.enabled
        break
      case 'lastFetchedAt':
        aValue = a.state?.lastFetchedAt || ''
        bValue = b.state?.lastFetchedAt || ''
        break
      case 'lastStatus':
        aValue = a.state?.lastStatus || ''
        bValue = b.state?.lastStatus || ''
        break
    }

    if (aValue === undefined || aValue === '') return 1
    if (bValue === undefined || bValue === '') return -1

    let comparison = 0
    if (typeof aValue === 'boolean' && typeof bValue === 'boolean') {
      comparison = aValue === bValue ? 0 : aValue ? -1 : 1
    } else if (aValue < bValue) {
      comparison = -1
    } else if (aValue > bValue) {
      comparison = 1
    }

    return sortOrder === 'asc' ? comparison : -comparison
  })

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
                <th onClick={() => handleSort('id')} className={styles.sortableHeader}>
                  ID{getSortIcon('id')}
                </th>
                <th onClick={() => handleSort('type')} className={styles.sortableHeader}>
                  Type{getSortIcon('type')}
                </th>
                <th>URL / Schema</th>
                <th>Tags</th>
                <th onClick={() => handleSort('enabled')} className={styles.sortableHeader}>
                  Enabled{getSortIcon('enabled')}
                </th>
                <th onClick={() => handleSort('lastFetchedAt')} className={styles.sortableHeader}>
                  Last Fetched{getSortIcon('lastFetchedAt')}
                </th>
                <th onClick={() => handleSort('lastStatus')} className={styles.sortableHeader}>
                  Status{getSortIcon('lastStatus')}
                </th>
              </tr>
            </thead>
            <tbody>
              {sortedSources.map((source) => (
                <tr
                  key={source.id}
                  onClick={(e) => handleRowClick(source.id, e)}
                  className={styles.clickableRow}
                >
                  <td>
                    <code className={styles.sourceId}>{source.id}</code>
                  </td>
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
                    {source.type === 'feed' && source.schema ? (
                      <code
                        className={styles.schema}
                        title={source.schemaDescription}
                      >
                        {source.schema}
                      </code>
                    ) : (
                      <a
                        href={source.url}
                        target="_blank"
                        rel="noopener noreferrer"
                        className={styles.url}
                      >
                        {source.url}
                      </a>
                    )}
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
