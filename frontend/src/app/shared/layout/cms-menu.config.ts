export interface CmsMenuItem {
  label: string;
  path: string;
}

export const CMS_MENU_ITEMS: CmsMenuItem[] = [
  { label: 'Dashboard', path: '/admin-cms' },
  { label: 'Quiz', path: '/admin-cms/quiz' },
  { label: 'Question & Answer', path: '/admin-cms/question-answer' },
  { label: 'Bank Soal', path: '/admin-cms/question-bank' },
  { label: 'Summary', path: '/admin-cms/summary' },
  { label: 'Analytics & Reporting', path: '/admin-cms/analytics' },
];
