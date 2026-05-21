# Neovim Cheatsheet & Features

## Core Features
- **Theme**: TokyoNight (Storm) with transparency.
- **Statusline**: Lualine with TokyoNight theme.
- **File Explorer**: Oil.nvim (Edit the filesystem like a normal buffer).
- **Terminal**: ToggleTerm (Floating terminal).
- **Fuzzy Finder**: Telescope.
- **Git**: Gitsigns for hunk management and status.
- **LSP Support**: Nix (`nixd`), Lua (`lua_ls`), Go (`gopls`), Rust (`rust_analyzer`), C/C++ (`clangd`).
- **AI Integration**: Gemini CLI.
- **Auto-format**: Automatically formats files on save for Nix, Lua, CSS, Go, Rust, and C/H.
- **Clipboard**: Synchronized with the system clipboard (`unnamedplus`).

## Keybindings
The **Leader** key is set to `Space`.

### Navigation & Files
| Key | Action |
|-----|--------|
| `-` | Open parent directory (Oil.nvim) |
| `<leader>ff` | Find files (Telescope) |
| `<leader>fg` | Live grep (Telescope) |
| `<C-h/j/k/l>` | Navigate between splits (works in terminal too) |

### Editing
| Key | Action |
|-----|--------|
| `<leader>w` | Save file |
| `<leader>q` | Quit editor |
| `]<Space>` | Add empty line below |
| `[<Space>` | Add empty line above |
| `<leader>p` | Paste below on a new line |
| `<leader>P` | Paste above on a new line |

### Autocompletion & Snippets
| Key | Action |
|-----|--------|
| `<C-Space>` | Trigger completion menu |
| `<CR>` | Confirm selection |
| `<Tab>` | Next item / Expand snippet / Jump forward |
| `<S-Tab>` | Previous item / Jump backward |
| `<C-e>` | Abort completion |

### LSP & Diagnostics
| Key | Action |
|-----|--------|
| `gd` | Go to definition |
| `K` | Show documentation (Hover) |
| `<leader>e` | Open floating diagnostics |
| `[d` | Go to previous diagnostic |
| `]d` | Go to next diagnostic |
| `<leader>rr` | Full Reload: Config + LSP |
| `<leader>rs` | Restart LSP only |

### Git (Gitsigns)
| Key | Action |
|-----|--------|
| `]c` | Next hunk |
| `[c` | Previous hunk |
| `<leader>hp` | Preview hunk |
| `<leader>hs` | Stage hunk |
| `<leader>hu` | Undo stage hunk |
| `<leader>hr` | Reset hunk |

### Terminal (ToggleTerm)
| Key | Action |
|-----|--------|
| `Ctrl-\` | Toggle floating terminal |
| `<esc>` | Exit terminal mode (to normal mode) |
----|--------|
| `<leader>ag` | Toggle Gemini Agent |
| `<leader>aq` | Ask Gemini (Visual mode selection) |
