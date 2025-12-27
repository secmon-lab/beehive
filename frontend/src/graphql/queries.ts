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
        lastError
        updatedAt
      }
    }
  }
`
