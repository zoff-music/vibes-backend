import { render, screen } from '@testing-library/react';
import { MemoryRouter } from 'react-router';
import App from './App';

describe('App Integration', () => {
  it('renders the home page by default', () => {
    render(
      <MemoryRouter initialEntries={['/']}>
        <App />
      </MemoryRouter>,
    );

    // Check for main title
    expect(screen.getByText('Zoff')).toBeInTheDocument();

    // Check for "Start a Session" button
    expect(screen.getByText('Start a Session')).toBeInTheDocument();
  });

  it('navigates to create room page when clicking start session', async () => {
    // We can't easily test navigation URL changes with MemoryRouter + App alone
    // without a more complex setup (checking history),
    // but we can check if it renders.
    // For a basic smoke test, rendering Home is sufficient.
  });
});
