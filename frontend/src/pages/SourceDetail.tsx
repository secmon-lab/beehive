import { useLazyQuery, useMutation, useQuery } from '@apollo/client'
import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { FETCH_SOURCE, GET_HISTORY, GET_SOURCE, LIST_HISTORIES } from '../graphql/queries'
import styles from './SourceDetail.module.css'

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

interface FetchError {
  message: string
  values: Array<{ key: string; value: string }>
}

interface History {
  id: string
  sourceID: string
  sourceType: string
  status: string
  startedAt: string
  completedAt: string
  processingTime: number
  urls: string[]
  itemsFetched: number
  ioCsExtracted: number
  ioCsCreated: number
  ioCsUpdated: number
  ioCsUnchanged: number
  errorCount: number
  errors: FetchError[]
  createdAt: string
}

interface GetSourceData {
  getSource: Source | null
}

interface ListHistoriesData {
  listHistories: {
    items: History[]
    total: number | null
  }
}

interface GetHistoryData {
  getHistory: History | null
}

interface FetchSourceData {
  fetchSource: History
}

function SourceDetail() {
  const { id } = useParams<{ id: string }>()
  const [pollingHistoryId, setPollingHistoryId] = useState<string | null>(null)

  const { loading: sourceLoading, error: sourceError, data: sourceData } = useQuery<GetSourceData>(
    GET_SOURCE,
    {
      variables: { id: id! },
      skip: !id,
    }
  )

  const {
    loading: historiesLoading,
    error: historiesError,
    data: historiesData,
    refetch: refetchHistories,
  } = useQuery<ListHistoriesData>(LIST_HISTORIES, {
    variables: { sourceID: id!, limit: 20, offset: 0 },
    skip: !id,
  })

  const [fetchSourceMutation, { loading: fetchLoading }] = useMutation<FetchSourceData>(
    FETCH_SOURCE,
    {
      onCompleted: (data) => {
        const history = data.fetchSource
        // Start polling if status is not completed
        if (history.status !== 'success' && history.status !== 'error') {
          setPollingHistoryId(history.id)
        } else {
          // If already completed, just refetch histories
          refetchHistories()
        }
      },
      onError: (error) => {
        console.error('Fetch error:', error)
      },
    }
  )

  const [getHistoryQuery, { stopPolling }] = useLazyQuery<GetHistoryData>(GET_HISTORY, {
    fetchPolicy: 'network-only',
    onCompleted: (data) => {
      if (data.getHistory) {
        // Stop polling if status is completed
        if (data.getHistory.status === 'success' || data.getHistory.status === 'error') {
          setPollingHistoryId(null)
          stopPolling()
          refetchHistories()
        }
      }
    },
  })

  // Polling effect
  useEffect(() => {
    if (!pollingHistoryId || !id) return

    // Start polling with 2 second interval
    const intervalId = setInterval(() => {
      getHistoryQuery({
        variables: {
          sourceID: id,
          id: pollingHistoryId,
        },
      })
    }, 2000)

    return () => {
      clearInterval(intervalId)
      stopPolling()
    }
  }, [pollingHistoryId, id, getHistoryQuery, stopPolling])

  const handleFetch = () => {
    if (!id) return
    fetchSourceMutation({ variables: { sourceID: id } })
  }

  if (!id) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>Source ID is missing</div>
        <Link to="/sources" className={styles.backLink}>
          ← Back to Source List
        </Link>
      </div>
    )
  }

  if (sourceLoading || historiesLoading) {
    return (
      <div className={styles.container}>
        <div className={styles.loading}>Loading source details...</div>
      </div>
    )
  }

  if (sourceError) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>Error loading source: {sourceError.message}</div>
      </div>
    )
  }

  if (historiesError) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>Error loading histories: {historiesError.message}</div>
      </div>
    )
  }

  const source = sourceData?.getSource

  if (!source) {
    return (
      <div className={styles.container}>
        <div className={styles.error}>Source not found</div>
        <Link to="/sources" className={styles.backLink}>
          ← Back to Source List
        </Link>
      </div>
    )
  }

  const histories = historiesData?.listHistories.items || []

  const getStatusBadgeClass = (status: string) => {
    switch (status) {
      case 'success':
        return styles.badgeSuccess
      case 'error':
        return styles.badgeError
      case 'running':
        return styles.badgeRunning
      default:
        return styles.badgeDefault
    }
  }

  return (
    <div className={styles.container}>
      <div className={styles.header}>
        <Link to="/sources" className={styles.backLink}>
          ← Back to Source List
        </Link>
        <h1 className={styles.title}>{source.id}</h1>
        <p className={styles.subtitle}>Source Details</p>
      </div>

      <div className={styles.card}>
        <h2 className={styles.sectionTitle}>Basic Information</h2>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>ID</div>
          <div className={styles.fieldValue}>{source.id}</div>
        </div>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>Type</div>
          <div className={styles.fieldValue}>{source.type}</div>
        </div>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>URL</div>
          <div className={styles.fieldValue}>
            <a href={source.url} target="_blank" rel="noopener noreferrer">
              {source.url}
            </a>
          </div>
        </div>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>Tags</div>
          <div className={styles.fieldValue}>
            {source.tags.length > 0 ? source.tags.join(', ') : '-'}
          </div>
        </div>

        <div className={styles.field}>
          <div className={styles.fieldLabel}>Enabled</div>
          <div className={styles.fieldValue}>
            <span
              className={`${styles.badge} ${
                source.enabled ? styles.badgeActive : styles.badgeInactive
              }`}
            >
              {source.enabled ? 'Yes' : 'No'}
            </span>
          </div>
        </div>
      </div>

      {source.state && (
        <div className={styles.card}>
          <h2 className={styles.sectionTitle}>Source State</h2>

          <div className={styles.field}>
            <div className={styles.fieldLabel}>Last Fetched</div>
            <div className={styles.fieldValue}>
              {source.state.lastFetchedAt
                ? new Date(source.state.lastFetchedAt).toLocaleString()
                : '-'}
            </div>
          </div>

          <div className={styles.field}>
            <div className={styles.fieldLabel}>Item Count</div>
            <div className={styles.fieldValue}>{source.state.itemCount}</div>
          </div>

          <div className={styles.field}>
            <div className={styles.fieldLabel}>Error Count</div>
            <div className={styles.fieldValue}>{source.state.errorCount}</div>
          </div>

          {source.state.lastError && (
            <div className={styles.field}>
              <div className={styles.fieldLabel}>Last Error</div>
              <div className={styles.fieldValue}>{source.state.lastError}</div>
            </div>
          )}
        </div>
      )}

      <div className={styles.card}>
        <div className={styles.fetchSection}>
          <h2 className={styles.sectionTitle}>Fetch History</h2>
          <div className={styles.fetchControls}>
            {pollingHistoryId && (
              <span className={styles.fetchingMessage}>Fetching...</span>
            )}
            <button
              onClick={handleFetch}
              disabled={fetchLoading || !!pollingHistoryId}
              className={styles.fetchButton}
            >
              Fetch Now
            </button>
          </div>
        </div>
        {histories.length === 0 ? (
          <p>No fetch history available</p>
        ) : (
          <table className={styles.table}>
            <thead>
              <tr>
                <th>Status</th>
                <th>Started At</th>
                <th>Completed At</th>
                <th>Processing Time</th>
                <th>Items Fetched</th>
                <th>IoCs Processed</th>
                <th>IoCs Created</th>
                <th>IoCs Updated</th>
                <th>Error Count</th>
              </tr>
            </thead>
            <tbody>
              {histories.map((history) => (
                <tr key={history.id}>
                  <td>
                    <span className={`${styles.badge} ${getStatusBadgeClass(history.status)}`}>
                      {history.status}
                    </span>
                  </td>
                  <td>{new Date(history.startedAt).toLocaleString()}</td>
                  <td>{new Date(history.completedAt).toLocaleString()}</td>
                  <td>{history.processingTime}ms</td>
                  <td>{history.itemsFetched}</td>
                  <td>{history.ioCsCreated + history.ioCsUpdated + history.ioCsUnchanged}</td>
                  <td>{history.ioCsCreated}</td>
                  <td>{history.ioCsUpdated}</td>
                  <td>{history.errorCount}</td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>
    </div>
  )
}

export default SourceDetail
