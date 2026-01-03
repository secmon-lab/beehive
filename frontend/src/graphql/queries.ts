import { gql } from '@apollo/client'

export const LIST_IOCS = gql`
  query ListIoCs($options: IoCListOptions) {
    listIoCs(options: $options) {
      items {
        id
        sourceID
        sourceType
        type
        value
        description
        sourceURL
        status
        firstSeenAt
        updatedAt
      }
      total
    }
  }
`

export const GET_IOC = gql`
  query GetIoC($id: ID!) {
    getIoC(id: $id) {
      id
      sourceID
      sourceType
      type
      value
      description
      sourceURL
      context
      status
      firstSeenAt
      updatedAt
    }
  }
`

export const LIST_SOURCES = gql`
  query ListSources {
    listSources {
      id
      type
      url
      tags
      enabled
      state {
        sourceID
        lastFetchedAt
        lastItemID
        lastItemDate
        itemCount
        errorCount
        lastStatus
        lastError
        updatedAt
      }
    }
  }
`

export const GET_SOURCE = gql`
  query GetSource($id: ID!) {
    getSource(id: $id) {
      id
      type
      url
      tags
      enabled
      state {
        sourceID
        lastFetchedAt
        lastItemID
        lastItemDate
        itemCount
        errorCount
        lastStatus
        lastError
        updatedAt
      }
    }
  }
`

export const LIST_HISTORIES = gql`
  query ListHistories($sourceID: String!, $limit: Int, $offset: Int) {
    listHistories(sourceID: $sourceID, limit: $limit, offset: $offset) {
      items {
        id
        sourceID
        sourceType
        status
        startedAt
        completedAt
        processingTime
        urls
        itemsFetched
        ioCsExtracted
        ioCsCreated
        ioCsUpdated
        ioCsUnchanged
        errorCount
        errors {
          message
          values {
            key
            value
          }
        }
        createdAt
      }
      total
    }
  }
`

export const GET_HISTORY = gql`
  query GetHistory($sourceID: String!, $id: ID!) {
    getHistory(sourceID: $sourceID, id: $id) {
      id
      sourceID
      sourceType
      status
      startedAt
      completedAt
      processingTime
      urls
      itemsFetched
      ioCsExtracted
      ioCsCreated
      ioCsUpdated
      ioCsUnchanged
      errorCount
      errors {
        message
        values {
          key
          value
        }
      }
      createdAt
    }
  }
`

export const FETCH_SOURCE = gql`
  mutation FetchSource($sourceID: String!) {
    fetchSource(sourceID: $sourceID) {
      id
      sourceID
      sourceType
      status
      startedAt
      completedAt
      processingTime
      urls
      itemsFetched
      ioCsExtracted
      ioCsCreated
      ioCsUpdated
      ioCsUnchanged
      errorCount
      errors {
        message
        values {
          key
          value
        }
      }
      createdAt
    }
  }
`
