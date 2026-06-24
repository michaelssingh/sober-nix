import express from 'express';
import path from 'path';
import { fileURLToPath } from 'url';
import cors from 'cors';

const __filename = fileURLToPath(import.meta.url);
const __dirname = path.dirname(__filename);

const app = express();
const PORT = process.env.PORT || 8080;

app.use(cors());
app.use(express.json());

// Serve static files from the 'dist' directory
app.use(express.static(path.join(__dirname, 'dist')));

// Mock API from backend/index.ts
const iacModules = [
  { id: 'vpc', name: 'AWS VPC', category: 'Networking', description: 'Production-grade VPC with public/private subnets and NAT Gateways.' },
  { id: 'eks', name: 'AWS EKS', category: 'Compute', description: 'Managed Kubernetes Cluster with OIDC, IRSA, and autoscaling node groups.' },
  { id: 'sober-edge-runtime', name: 'SoberEdge Runtime', category: 'Serverless', description: 'The eBPF-enabled WASM runtime for high-performance functions.' }
];

let deployments = [
  { id: 'dep-01', moduleId: 'vpc', status: 'Healthy', lastApplied: new Date().toISOString() }
];

app.get('/api/modules', (req, res) => {
  res.json(iacModules);
});

app.get('/api/deployments', (req, res) => {
  res.json(deployments);
});

app.post('/api/deploy', (req, res) => {
  const { moduleId } = req.body;
  const newDeployment = {
    id: `dep-${Math.random().toString(36).substr(2, 4)}`,
    moduleId,
    status: 'In Progress',
    lastApplied: new Date().toISOString(),
    logs: [
      `Initializing ${moduleId}...`,
      `Analyzing eBPF constraints...`,
      `Compiling WASM bytecode...`,
      `Deployment successful.`
    ]
  };
  deployments.push(newDeployment);
  res.status(201).json(newDeployment);
});

// Handle React routing, return all requests to index.html
app.get('*', (req, res) => {
  res.sendFile(path.join(__dirname, 'dist', 'index.html'));
});

app.listen(PORT, '0.0.0.0', () => {
  console.log(`Server running on http://0.0.0.0:${PORT}`);
});
