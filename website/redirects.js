// REDIRECTS FILE

// See the README file in this directory for documentation. Please do not
// modify or delete existing redirects without first verifying internally.
// Next.js redirect documentation: https://nextjs.org/docs/api-reference/next.config.js/redirects

module.exports = [
  {
    source: '/docs/connect/cluster-peering/create-manage-peering',
    destination:
      '/docs/connect/cluster-peering/usage/establish-cluster-peering',
    permanent: true,
  },
  {
    source: '/docs/connect/cluster-peering/k8s',
    destination: '/docs/k8s/connect/cluster-peering/k8s-tech-specs',
    permanent: true,
  },
]
