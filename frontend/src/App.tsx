import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import Layout from './components/Layout'
import IoCList from './pages/IoCList'
import IoCDetail from './pages/IoCDetail'
import SourceList from './pages/SourceList'
import SourceDetail from './pages/SourceDetail'

function App() {
  return (
    <BrowserRouter>
      <Layout>
        <Routes>
          <Route path="/" element={<Navigate to="/ioc" replace />} />
          <Route path="/ioc" element={<IoCList />} />
          <Route path="/ioc/:id" element={<IoCDetail />} />
          <Route path="/sources" element={<SourceList />} />
          <Route path="/sources/:id" element={<SourceDetail />} />
        </Routes>
      </Layout>
    </BrowserRouter>
  )
}

export default App
