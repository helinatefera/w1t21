import { useState } from 'react';
import { useQuery } from '@tanstack/react-query';
import { Link } from 'react-router-dom';
import { listCollectibles } from '../api/collectibles';
import { formatCents } from '../utils/formatters';
import { useAuthStore } from '../store/authStore';
import { useABStore } from '../store/abStore';

export function CatalogPage() {
  const [page, setPage] = useState(1);
  const { hasRole } = useAuthStore();

  const catalogVariant = useABStore((s) => s.getVariant('catalog_layout'));
  const searchRanking = useABStore((s) => s.getVariant('search_ranking'));

  const { data, isLoading } = useQuery({
    queryKey: ['collectibles', page],
    queryFn: () => listCollectibles(page).then((r) => r.data),
  });

  // search_ranking experiment: "popular" sorts by view_count descending
  const sortedItems = data?.data
    ? searchRanking === 'popular'
      ? [...data.data].sort((a, b) => b.view_count - a.view_count)
      : data.data
    : [];

  return (
    <div>
      <div className="flex justify-between items-center mb-4 md:mb-6">
        <h1 className="text-xl md:text-2xl font-bold">Catalog</h1>
        {hasRole('seller') && (
          <Link
            to="/sell/new"
            className="px-4 py-2 bg-primary-600 text-white rounded text-sm hover:bg-primary-700 transition-colors"
          >
            + Add Collectible
          </Link>
        )}
      </div>

      {isLoading ? (
        <p className="text-gray-500">Loading...</p>
      ) : (
        <>
          {/* Responsive grid: 1 col tablet-portrait, 2 cols tablet-landscape, 3-4 cols desktop */}
          <div className={`grid gap-4 md:gap-5 ${
              catalogVariant === 'list'
                ? 'grid-cols-1'
                : 'grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4'
            }`}>
            {sortedItems.map((item) => (
              <Link
                key={item.id}
                to={`/catalog/${item.id}`}
                className="bg-white p-3 md:p-4 rounded-lg shadow-sm border hover:shadow-md transition-shadow"
              >
                {item.image_url ? (
                  <img src={item.image_url} alt={item.title} className="w-full h-40 md:h-48 object-cover rounded mb-3" />
                ) : (
                  <div className="w-full h-40 md:h-48 bg-gray-100 rounded mb-3 flex items-center justify-center text-gray-400">
                    No image
                  </div>
                )}
                <h3 className="font-medium text-gray-900 truncate">{item.title}</h3>
                <p className="text-sm text-gray-500 mt-1 line-clamp-2">{item.description}</p>
                <div className="flex justify-between items-center mt-3">
                  <p className="text-base md:text-lg font-bold text-primary-700">
                    {formatCents(item.price_cents, item.currency)}
                  </p>
                  <span className="text-xs text-gray-400">{item.view_count} views</span>
                </div>
              </Link>
            ))}
          </div>

          {data && data.total_pages > 1 && (
            <div className="flex justify-center items-center gap-2 mt-6">
              <button
                onClick={() => setPage((p) => Math.max(1, p - 1))}
                disabled={page <= 1}
                className="px-3 md:px-4 py-2 bg-white border rounded text-sm disabled:opacity-50"
              >
                Previous
              </button>
              <span className="px-3 md:px-4 py-2 text-sm text-gray-600">
                Page {page} of {data.total_pages}
              </span>
              <button
                onClick={() => setPage((p) => p + 1)}
                disabled={page >= data.total_pages}
                className="px-3 md:px-4 py-2 bg-white border rounded text-sm disabled:opacity-50"
              >
                Next
              </button>
            </div>
          )}
        </>
      )}
    </div>
  );
}
