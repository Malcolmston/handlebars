import { describe, it, expect, vi, beforeEach } from 'vitest';
import { render, screen } from '@testing-library/react';
import { DocsView } from '../../../src/components/DocsView';
import type { DocIndex } from 'go-ui';

// A minimal DocIndex the stubbed fetch returns for DocsApp's doc.json request.
const DOC_INDEX: DocIndex = {
  module: 'github.com/malcolmston/handlebars',
  packages: [
    {
      importPath: 'github.com/malcolmston/handlebars',
      name: 'handlebars',
      synopsis: 'Package handlebars is a dependency-free Handlebars/Mustache templating engine in pure Go.',
      doc: 'Package handlebars is a dependency-free Handlebars/Mustache templating engine in pure Go.',
      consts: [],
      vars: [],
      types: [
        {
          name: 'Template',
          signature: 'type Template struct{}',
          doc: 'Template is a compiled Handlebars template with its own helper and partial registry.',
          consts: [],
          vars: [],
          funcs: [],
          methods: [],
        },
      ],
      funcs: [{ name: 'Render', signature: 'func Render(source string, data interface{}) (string, error)', doc: 'Render parses source and renders it against data in one call.' }],
    },
  ],
};

describe('DocsView', () => {
  beforeEach(() => {
    // DocsApp fetches doc.json; return the small index.
    global.fetch = vi.fn((input: RequestInfo | URL) => {
      if (String(input).includes('doc.json')) {
        return Promise.resolve({ ok: true, json: () => Promise.resolve(DOC_INDEX) } as Response);
      }
      return new Promise<Response>(() => {});
    }) as unknown as typeof fetch;
  });

  it('renders the inline React API reference from the fetched doc.json', async () => {
    const { container } = render(<DocsView />);
    expect(container.querySelector('#view-docs')).not.toBeNull();
    expect(
      screen.getByRole('heading', { level: 2, name: /API documentation/ }),
    ).toBeInTheDocument();

    // DocsApp fetches asynchronously, then renders the package view + symbols.
    expect(await screen.findByRole('heading', { name: /package handlebars/ })).toBeInTheDocument();
    expect(container.querySelector('#sym-Render'), 'func Render symbol card').not.toBeNull();
    expect(container.querySelector('#sym-Template'), 'type Template symbol card').not.toBeNull();

    // The secondary link to the raw generated static HTML remains.
    expect(screen.getByRole('link', { name: /Open the raw generated HTML/ })).toHaveAttribute('href', './api/');
  });
});
