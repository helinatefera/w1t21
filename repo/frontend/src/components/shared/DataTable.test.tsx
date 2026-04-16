// @vitest-environment happy-dom
import { describe, it, expect, vi } from 'vitest';
import { render, screen, fireEvent } from '@testing-library/react';
import { DataTable } from './DataTable';

describe('DataTable component', () => {
  const columns = [
    { key: 'name', header: 'Name' },
    { key: 'status', header: 'Status' },
  ];

  const data = [
    { name: 'Item A', status: 'active' },
    { name: 'Item B', status: 'inactive' },
  ];

  it('renders column headers', () => {
    render(<DataTable columns={columns} data={data} page={1} totalPages={1} onPageChange={() => {}} />);
    expect(screen.getByText('Name')).toBeDefined();
    expect(screen.getByText('Status')).toBeDefined();
  });

  it('renders data rows', () => {
    render(<DataTable columns={columns} data={data} page={1} totalPages={1} onPageChange={() => {}} />);
    expect(screen.getByText('Item A')).toBeDefined();
    expect(screen.getByText('Item B')).toBeDefined();
    expect(screen.getByText('active')).toBeDefined();
  });

  it('hides pagination when totalPages is 1', () => {
    render(<DataTable columns={columns} data={data} page={1} totalPages={1} onPageChange={() => {}} />);
    expect(screen.queryByText('Previous')).toBeNull();
    expect(screen.queryByText('Next')).toBeNull();
  });

  it('shows pagination when totalPages > 1', () => {
    render(<DataTable columns={columns} data={data} page={1} totalPages={3} onPageChange={() => {}} />);
    expect(screen.getByText('Previous')).toBeDefined();
    expect(screen.getByText('Next')).toBeDefined();
    expect(screen.getByText('Page 1 of 3')).toBeDefined();
  });

  it('calls onPageChange when Next is clicked', () => {
    const onPageChange = vi.fn();
    render(<DataTable columns={columns} data={data} page={1} totalPages={3} onPageChange={onPageChange} />);
    fireEvent.click(screen.getByText('Next'));
    expect(onPageChange).toHaveBeenCalledWith(2);
  });

  it('calls onPageChange when Previous is clicked', () => {
    const onPageChange = vi.fn();
    render(<DataTable columns={columns} data={data} page={2} totalPages={3} onPageChange={onPageChange} />);
    fireEvent.click(screen.getByText('Previous'));
    expect(onPageChange).toHaveBeenCalledWith(1);
  });

  it('disables Previous on first page', () => {
    render(<DataTable columns={columns} data={data} page={1} totalPages={3} onPageChange={() => {}} />);
    const prev = screen.getByText('Previous') as HTMLButtonElement;
    expect(prev.disabled).toBe(true);
  });

  it('disables Next on last page', () => {
    render(<DataTable columns={columns} data={data} page={3} totalPages={3} onPageChange={() => {}} />);
    const next = screen.getByText('Next') as HTMLButtonElement;
    expect(next.disabled).toBe(true);
  });

  it('renders custom column renderer', () => {
    const customColumns = [
      { key: 'name', header: 'Name', render: (item: any) => `Custom: ${item.name}` },
    ];
    render(<DataTable columns={customColumns} data={data} page={1} totalPages={1} onPageChange={() => {}} />);
    expect(screen.getByText('Custom: Item A')).toBeDefined();
  });

  it('renders empty table with no data', () => {
    render(<DataTable columns={columns} data={[]} page={1} totalPages={1} onPageChange={() => {}} />);
    expect(screen.getByText('Name')).toBeDefined();
    expect(screen.queryByText('Item A')).toBeNull();
  });
});
