export type CmsMenuIcon =
  | 'dashboard'
  | 'quiz'
  | 'question-answer'
  | 'question-bank'
  | 'summary'
  | 'analytics'
  | 'collaboration-permission';

export interface CmsMenuItem {
  label: string;
  path: string;
  icon: CmsMenuIcon;
  roles?: Array<'super_admin' | 'editor'>;
}

export const CMS_MENU_ITEMS: CmsMenuItem[] = [
  { label: 'Dashboard', path: '/admin-cms', icon: 'dashboard' },
  { label: 'Quiz', path: '/admin-cms/quiz', icon: 'quiz' },
  { label: 'Question & Answer', path: '/admin-cms/question-answer', icon: 'question-answer' },
  { label: 'Bank Soal', path: '/admin-cms/question-bank', icon: 'question-bank' },
  { label: 'Summary', path: '/admin-cms/summary', icon: 'summary' },
  { label: 'Analytics & Reporting', path: '/admin-cms/analytics', icon: 'analytics' },
  {
    label: 'Kolaborasi & Permission',
    path: '/admin-cms/collaboration-permission',
    icon: 'collaboration-permission',
  },
];
