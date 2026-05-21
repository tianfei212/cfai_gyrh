export const BACKGROUND_MANAGER_PAGE_SIZE = 10;

export function getSafePage(page) {
  return Number.isFinite(page) && page > 1 ? Math.floor(page) : 1;
}

export function getTotalPages(total, limit = BACKGROUND_MANAGER_PAGE_SIZE) {
  const safeTotal = Number.isFinite(total) && total > 0 ? total : 0;
  const safeLimit = Number.isFinite(limit) && limit > 0 ? limit : BACKGROUND_MANAGER_PAGE_SIZE;
  return Math.max(1, Math.ceil(safeTotal / safeLimit));
}

export function buildBackgroundPromptListUrl(page, limit = BACKGROUND_MANAGER_PAGE_SIZE, { categoryId = 0 } = {}) {
  const safePage = getSafePage(page);
  const safeLimit = Number.isFinite(limit) && limit > 0 ? Math.floor(limit) : BACKGROUND_MANAGER_PAGE_SIZE;
  const offset = (safePage - 1) * safeLimit;
  const params = new URLSearchParams({
    limit: String(safeLimit),
    offset: String(offset),
  });
  if (Number.isFinite(categoryId) && categoryId > 0) {
    params.set('category_id', String(Math.floor(categoryId)));
  }
  return `/api/v1/background-prompts?${params.toString()}`;
}

export function getPageAfterRefresh(currentPage, { resetToFirstPage = false } = {}) {
  return resetToFirstPage ? 1 : getSafePage(currentPage);
}
