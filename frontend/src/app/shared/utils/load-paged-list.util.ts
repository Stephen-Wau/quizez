import { Observable } from 'rxjs';
import { ToastService } from '../ui/toast/toast.service';
import { PagedResult } from '../ui/data-table/data-table.component';

// Pola load list standar semua menu CRUD CMS yang pakai <app-data-table serverSide>: hit API list
// (dibungkus {data, meta} dari BE), lalu sync rows/totalCount/pageSize ke state komponen, atau
// toast error kalau gagal. Generic <T> biar dipakai model apa pun (WorkHistory, Education, dst).
export function loadPagedList<T>(
  request: Observable<PagedResult<T>>,
  toast: ToastService,
  errorMessage: string,
  onSuccess: (data: T[], totalCount: number, pageSize: number) => void,
): void {
  request.subscribe({
    next: ({ data, meta }) => onSuccess(data, meta.total, meta.per_page),
    error: () => toast.error(errorMessage),
  });
}
