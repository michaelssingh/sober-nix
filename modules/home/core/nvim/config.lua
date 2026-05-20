-- SYNC CLIPBOARD
-- 'unnamedplus' allows you to copy/paste between Neovim and other apps
vim.opt.clipboard = "unnamedplus"

-- 1. Basics & Theme
-- 1. Theme Configuration (The Transparency Fix)
require("tokyonight").setup({
  style = "storm",    -- The specific palette you like
  transparent = true, -- <--- THIS IS THE KEY
  styles = {
    sidebars = "transparent",
    floats = "transparent",
  },
})
vim.cmd [[colorscheme tokyonight]]

vim.g.mapleader = " "
vim.opt.number = true
vim.opt.relativenumber = true
vim.opt.termguicolors = true -- True color support
vim.opt.scrolloff = 8        -- Keep cursor away from edge
vim.opt.cursorline = true

vim.cmd [[
	highlight Normal guibg=none
	highlight NonText guibg=none
	highlight Normal ctermbg=none
	highlight NonText ctermbg=none
]]

vim.opt.foldmethod = "expr"
vim.opt.foldexpr = "nvim_treesitter#foldexpr()"
vim.opt.foldenable = false -- Don't fold everything by default when opening a file

-- 2. Global Indentation (Nix/Lua standard: 2 spaces)
vim.opt.expandtab = true
vim.opt.shiftwidth = 2
vim.opt.softtabstop = 2
vim.opt.tabstop = 2
vim.opt.smartindent = true

-- 3. Search Logic
vim.opt.ignorecase = true
vim.opt.smartcase = true

-- 4. Plugin Setup
require('lualine').setup { options = { theme = 'tokyonight' } }
require('gitsigns').setup({
  on_attach = function(bufnr)
    local gs = package.loaded.gitsigns

    local function map(mode, l, r, opts)
      opts = opts or {}
      opts.buffer = bufnr
      vim.keymap.set(mode, l, r, opts)
    end

    -- Navigation (Jump between changes)
    map('n', ']c', function()
      if vim.wo.diff then return ']c' end
      vim.schedule(function() gs.next_hunk() end)
      return '<Ignore>'
    end, { expr = true, desc = "Next Git Hunk" })

    map('n', '[c', function()
      if vim.wo.diff then return '[c' end
      vim.schedule(function() gs.prev_hunk() end)
      return '<Ignore>'
    end, { expr = true, desc = "Prev Git Hunk" })

    -- Actions
    map('n', '<leader>hp', gs.preview_hunk, { desc = "Preview Hunk" }) -- Floating window of the change
    map('n', '<leader>hs', gs.stage_hunk, { desc = "Stage Hunk" })     -- "git add" just this chunk
    map('n', '<leader>hu', gs.undo_stage_hunk, { desc = "Undo Stage" })
    map('n', '<leader>hr', gs.reset_hunk, { desc = "Reset Hunk" })     -- Revert this chunk
  end
})
require('which-key').setup()
require('nvim-autopairs').setup({})

-- Oil.nvim: Edit file system like a buffer
require("oil").setup({
  default_file_explorer = true,
  columns = { "icon", "permissions", "size", "mtime" },
  view_options = {
    show_hidden = true,
  },
})

-- Map '-' to open the parent directory (very intuitive)
vim.keymap.set("n", "-", "<CMD>Oil<CR>", { desc = "Open Parent Directory" })

-- TOGGLETERM SETUP
require("toggleterm").setup {
  size = 20,
  open_mapping = [[<c-\>]], -- The key to toggle the terminal (Ctrl + Backslash)
  hide_numbers = true,
  shade_filetypes = {},
  shade_terminals = true,
  shading_factor = 2,
  start_in_insert = true,
  insert_mappings = true,
  persist_size = true,
  direction = "float", -- Options: 'vertical' | 'horizontal' | 'tab' | 'float'
  close_on_exit = true,
  shell = vim.o.shell,
  float_opts = {
    border = "curved",
    winblend = 0,
    highlights = {
      border = "Normal",
      background = "Normal",
    },
  },
}

-- KEYMAPPING: Better Navigation
-- This lets you navigate OUT of the terminal window using Ctrl+h/j/k/l
-- just like you move between code splits.
function _G.set_terminal_keymaps()
  local opts = { buffer = 0 }
  vim.keymap.set('t', '<esc>', [[<C-\><C-n>]], opts)
  vim.keymap.set('t', '<C-h>', [[<Cmd>wincmd h<CR>]], opts)
  vim.keymap.set('t', '<C-j>', [[<Cmd>wincmd j<CR>]], opts)
  vim.keymap.set('t', '<C-k>', [[<Cmd>wincmd k<CR>]], opts)
  vim.keymap.set('t', '<C-l>', [[<Cmd>wincmd l<CR>]], opts)
end

-- Apply these mappings only when a terminal is open
vim.cmd('autocmd! TermOpen term://* lua set_terminal_keymaps()')

require("ibl").setup {
  scope = { enabled = true, show_start = true, show_end = false },
  indent = { char = "▏" },
}

require('nvim-treesitter.configs').setup({
  -- Add these three lines to satisfy the LSP's type checking
  ensure_installed = {},
  sync_install = false,
  auto_install = false,
  modules = {},

  ignore_install = {}, -- Also helpful to keep the LSP happy

  highlight = {
    enable = true,
  },
  indent = {
    enable = true
  },
})

-- 5. LSP & Autocompletion Setup
local lspconfig = require('lspconfig')
local cmp = require('cmp')
local luasnip = require('luasnip')

-- Add capabilities for nvim-cmp
local capabilities = require('cmp_nvim_lsp').default_capabilities()

-- Autocompletion Setup
cmp.setup({
  snippet = {
    expand = function(args)
      luasnip.lsp_expand(args.body)
    end,
  },
  mapping = cmp.mapping.preset.insert({
    ['<C-b>'] = cmp.mapping.scroll_docs(-4),
    ['<C-f>'] = cmp.mapping.scroll_docs(4),
    ['<C-Space>'] = cmp.mapping.complete(),
    ['<C-e>'] = cmp.mapping.abort(),
    ['<CR>'] = cmp.mapping.confirm({ select = true }), -- Accept currently selected item. Set `select` to `false` to only confirm explicitly selected items.
    ['<Tab>'] = cmp.mapping(function(fallback)
      if cmp.visible() then
        cmp.select_next_item()
      elseif luasnip.expand_or_jumpable() then
        luasnip.expand_or_jump()
      else
        fallback()
      end
    end, { 'i', 's' }),
    ['<S-Tab>'] = cmp.mapping(function(fallback)
      if cmp.visible() then
        cmp.select_prev_item()
      elseif luasnip.jumpable(-1) then
        luasnip.jump(-1)
      else
        fallback()
      end
    end, { 'i', 's' }),
  }),
  sources = cmp.config.sources({
    { name = 'nvim_lsp' },
    { name = 'luasnip' },
  }, {
    { name = 'buffer' },
    { name = 'path' },
  })
})

-- Go Setup (Added for SOBER VPN Manager)
lspconfig.gopls.setup({
  capabilities = capabilities,
  settings = {
    gopls = {
      analyses = {
        unusedparams = true,
      },
      staticcheck = true,
      gofumpt = true,
    },
  },
})

-- Rust Setup
lspconfig.rust_analyzer.setup({
  capabilities = capabilities,
})

-- C/C++ Setup
lspconfig.clangd.setup({
  capabilities = capabilities,
})

-- Clean Lua Setup
lspconfig.lua_ls.setup({
  capabilities = capabilities,
  settings = {
    Lua = {
      runtime = { version = 'LuaJIT' },
      diagnostics = { globals = { 'vim' } },
      workspace = {
        library = vim.api.nvim_get_runtime_file("", true),
        checkThirdParty = false,
      },
      telemetry = { enable = false },
    },
  },
})

-- Clean Nix Setup
lspconfig.nixd.setup({
  capabilities = capabilities,
  settings = {
    nixd = {
      formatting = { command = { "nixfmt" } },
    },
  },
})

-- 6. Diagnostics UI
vim.diagnostic.config({
  virtual_text = { prefix = '●', spacing = 4 },
  signs = true,
  underline = true,
  update_in_insert = false,
  severity_sort = true,
})

local signs = { Error = "󰅚 ", Warn = "󰀪 ", Hint = "󰌶 ", Info = "➔ " }
for type, icon in pairs(signs) do
  local hl = "DiagnosticSign" .. type
  vim.fn.sign_define(hl, { text = icon, texthl = hl, numhl = hl })
end

-- 7. Keybindings & Plugin Extensions
-- Telescope Setup
require('telescope').setup({
  extensions = {
    fzf = {
      fuzzy = true,
      override_generic_sorter = true,
      override_file_sorter = true,
      case_mode = "smart_case",
    }
  }
})
require('telescope').load_extension('fzf')

-- Gemini Setup
require('gemini').setup({})

local builtin = require('telescope.builtin')
vim.keymap.set('n', '<leader>ff', builtin.find_files, { desc = "Find Files" })
vim.keymap.set('n', '<leader>fg', builtin.live_grep, { desc = "Live Grep" })
vim.keymap.set('n', 'gd', vim.lsp.buf.definition, { desc = "Go to Definition" })
vim.keymap.set('n', 'K', vim.lsp.buf.hover, { desc = "Docs" })
vim.keymap.set('n', '<leader>e', vim.diagnostic.open_float, { desc = "Line Diagnostics" })
vim.keymap.set('n', '[d', vim.diagnostic.goto_prev, { desc = "Prev Diagnostic" })
vim.keymap.set('n', ']d', vim.diagnostic.goto_next, { desc = "Next Diagnostic" })

-- RELOAD & RESTART (The "Full Refresh")
vim.keymap.set("n", "<leader>rr", function()
  -- 1. Clear loaded Lua modules (forces re-read)
  for name, _ in pairs(package.loaded) do
    if name:match("^user") then
      package.loaded[name] = nil
    end
  end

  -- 2. Reload the config
  dofile(vim.env.MYVIMRC)

  -- 3. Restart the LSP (The new logic)
  -- This makes sure any changes to lspconfig are applied immediately
  vim.cmd("LspRestart")

  -- 4. Notify
  vim.notify("Config Reloaded & LSP Restarted!", vim.log.levels.INFO)
end, { desc = "Reload Config + LSP" })

-- Standalone Restart (If you just want to reboot the server without reloading config)
vim.keymap.set("n", "<leader>rs", ":LspRestart<CR>", { desc = "Restart LSP" })
-- 8. Autocommands (Auto-format)
vim.api.nvim_create_autocmd("BufWritePre", {
  pattern = { "*.nix", "*.lua", "*.css", "*.go", "*.rs", "*.c", "*.h" },
  callback = function()
    vim.lsp.buf.format()
  end,
})
-- Fast Saving
vim.keymap.set("n", "<leader>w", ":w<CR>", { desc = "Save File" })

-- Fast Quitting
vim.keymap.set("n", "<leader>q", ":q<CR>", { desc = "Exit editor" })

-- INSERT EMPTY LINES (No Insert Mode)
-- ]<Space>  = Add line below
-- [<Space>  = Add line above

vim.keymap.set("n", "]<Space>", function()
  vim.fn.append(vim.fn.line("."), "")
end, { desc = "Add Empty Line Below" })

vim.keymap.set("n", "[<Space>", function()
  vim.fn.append(vim.fn.line(".") - 1, "")
end, { desc = "Add Empty Line Above" })

-- SMART PASTE (Forces content onto a new line)
-- This solves the "Shift+P" frustration by using the :put command
-- which treats all text as linewise.
-- <leader>p = Paste on a new line BELOW
vim.keymap.set("n", "<leader>p", ":pu<CR>", { desc = "Paste Below (Force Line)" })
-- <leader>P = Paste on a new line ABOVE
vim.keymap.set("n", "<leader>P", ":pu!<CR>", { desc = "Paste Above (Force Line)" })
-- Gemini CLI Toggle
vim.keymap.set('n', '<leader>ag', '<cmd>Gemini toggle<cr>', { desc = "Toggle Gemini Agent" })
vim.keymap.set('v', '<leader>aq', '<cmd>Gemini ask<cr>', { desc = "Ask Gemini [Query]" })
