import { useQuery, useInfiniteQuery } from "@tanstack/react-query";
import apiClient from "@/lib/api-client";
import type { ListIdeasResponse, ListIdeasParams, IdeaDetail } from "@/types/api";

export function useIdeas(params: ListIdeasParams = {}) {
  return useQuery<ListIdeasResponse>({
    queryKey: ["ideas", params],
    queryFn: async () => {
      const res = await apiClient.get("/api/v1/ideas", { params });
      return res.data;
    },
  });
}

export function useInfiniteIdeas(params: Omit<ListIdeasParams, "page"> & { page_size?: number } = {}) {
  const pageSize = params.page_size || 20;
  return useInfiniteQuery<ListIdeasResponse>({
    queryKey: ["ideas-infinite", { ...params, page_size: pageSize }],
    queryFn: async ({ pageParam }) => {
      const res = await apiClient.get("/api/v1/ideas", {
        params: { ...params, page: pageParam, page_size: pageSize },
      });
      return res.data;
    },
    initialPageParam: 1,
    getNextPageParam: (lastPage, allPages) => {
      const loaded = allPages.reduce((sum, p) => sum + p.items.length, 0);
      return loaded < lastPage.total ? allPages.length + 1 : undefined;
    },
    staleTime: 2 * 60 * 1000,
    gcTime: 5 * 60 * 1000,
  });
}

export function useIdea(id: number) {
  return useQuery<IdeaDetail>({
    queryKey: ["idea", id],
    queryFn: async () => {
      const res = await apiClient.get(`/api/v1/ideas/${id}`);
      return res.data;
    },
    enabled: !!id,
  });
}
