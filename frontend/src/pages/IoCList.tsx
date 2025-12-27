import { useState } from 'react'
import { useQuery } from '@apollo/client'
import { useNavigate } from 'react-router-dom'
import { LIST_IOCS } from '../graphql/queries'
import styles from './IoCList.module.css'

interface IoC {
  id: string
  sourceID: string
  sourceType: string
  type: string
  value: string
  description: string
  sourceURL?: string
  status: string
  firstSeenAt: string
  updatedAt: string
}

interface IoCConnection {
  items: IoC[]
  total: number
}

interface ListIoCsData {
  listIoCs: IoCConnection
}

// TODO: Use GraphQL Code Generator to auto-generate these types from schema
// These must match the GraphQL schema enums: IoCSortField and SortOrder
type SortField = 'TYPE' | 'VALUE' | 'SOURCE_ID' | 'STATUS' | 'FIRST_SEEN_AT' | 'UPDATED_AT'
type SortOrder = 'ASC' | 'DESC'

function IoCList() {
  const navigate = useNavigate()
  const [pageSize, setPageSize] = useState(20)
  const [page, setPage] = useState(0)
  const [sortField, setSortField] = useState<SortField>('UPDATED_AT')
  const [sortOrder, setSortOrder] = useState<SortOrder>('DESC')

  const { loading, error, data } = useQuery<ListIoCsData>(LIST_IOCS, {
    variables: {
      options: {
        offset: page * pageSize,
        limit: pageSize,
        sortField,
        sortOrder,
      },
    },
  })

  if (loading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading IoCs...</div>
      </div>
    )
  }

  if (error) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>Error loading IoCs: {error.message}</div>
      </div>
    )
  }

  const connection = data?.listIoCs
  const iocs = connection?.items || []
  const total = connection?.total || 0
  const totalPages = Math.ceil(total / pageSize)

  const handleSort = (field: SortField) => {
    if (sortField === field) {
      setSortOrder(sortOrder === 'ASC' ? 'DESC' : 'ASC')
    } else {
      setSortField(field)
      setSortOrder('ASC')
    }
    setPage(0) // Reset to first page when sorting changes
  }

  const getSortIcon = (field: SortField) => {
    if (sortField !== field) return ' ↕'
    return sortOrder === 'ASC' ? ' ↑' : ' ↓'
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <h1 className={styles.title}>Indicators of Compromise</h1>
        <p className={styles.subtitle}>
          Showing {iocs.length} of {total} IoCs
        </p>
      </div>

      <div className={styles.controls}>
        <div className={styles.pageSizeSelector}>
          <label htmlFor="pageSize">Items per page:</label>
          <select
            id="pageSize"
            value={pageSize}
            onChange={(e) => {
              setPageSize(Number(e.target.value))
              setPage(0)
            }}
            className={styles.select}
          >
            <option value={20}>20</option>
            <option value={50}>50</option>
            <option value={100}>100</option>
          </select>
        </div>

        {totalPages > 1 && (
          <div className={styles.pagination}>
            <button
              onClick={() => setPage(Math.max(0, page - 1))}
              disabled={page === 0}
              className={styles.paginationButton}
            >
              Previous
            </button>
            <span className={styles.pageInfo}>
              Page {page + 1} of {totalPages}
            </span>
            <button
              onClick={() => setPage(Math.min(totalPages - 1, page + 1))}
              disabled={page >= totalPages - 1}
              className={styles.paginationButton}
            >
              Next
            </button>
          </div>
        )}
      </div>

      {iocs.length === 0 ? (
        <div className={styles.empty}>No IoCs found</div>
      ) : (
        <div className={styles.tableContainer}>
          <table className={styles.table}>
            <thead>
              <tr>
                <th onClick={() => handleSort('TYPE')} className={styles.sortable}>
                  Type{getSortIcon('TYPE')}
                </th>
                <th onClick={() => handleSort('VALUE')} className={styles.sortable}>
                  Value{getSortIcon('VALUE')}
                </th>
                <th>Description</th>
                <th onClick={() => handleSort('SOURCE_ID')} className={styles.sortable}>
                  Source{getSortIcon('SOURCE_ID')}
                </th>
                <th onClick={() => handleSort('STATUS')} className={styles.sortable}>
                  Status{getSortIcon('STATUS')}
                </th>
                <th onClick={() => handleSort('FIRST_SEEN_AT')} className={styles.sortable}>
                  First Seen{getSortIcon('FIRST_SEEN_AT')}
                </th>
                <th onClick={() => handleSort('UPDATED_AT')} className={styles.sortable}>
                  Updated{getSortIcon('UPDATED_AT')}
                </th>
              </tr>
            </thead>
            <tbody>
              {iocs.map((ioc) => (
                <tr key={ioc.id} onClick={() => navigate(`/ioc/${ioc.id}`)} className={styles.clickableRow}>
                  <td>{ioc.type}</td>
                  <td className={styles.valueCell}>{ioc.value}</td>
                  <td>{ioc.description || '-'}</td>
                  <td>{ioc.sourceID}</td>
                  <td>
                    <span
                      className={`${styles.badge} ${
                        ioc.status === 'active' ? styles.badgeActive : styles.badgeInactive
                      }`}
                    >
                      {ioc.status}
                    </span>
                  </td>
                  <td>{new Date(ioc.firstSeenAt).toLocaleString()}</td>
                  <td>{new Date(ioc.updatedAt).toLocaleString()}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  )
}

export default IoCList
