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

-- 1. Core Treesitter Configuration
require('nvim-treesitter.config').setup({
  highlight = { enable = true },
  indent = { enable = true },
})

-- 2. Treesitter Textobjects Configuration
require("nvim-treesitter-textobjects").setup({
  select = {
    enable = true,
    lookahead = true, -- Automatically jump forward to the next textobject
    keymaps = {
      ["af"] = "@function.outer",
      ["if"] = "@function.inner",
      ["ac"] = "@class.outer",
      ["ic"] = "@class.inner",
    },
  },
})

-- 5. LSP & Autocompletion Setup
local cmp = require('cmp')
local luasnip = require('luasnip')

-- Centralized LSP Configuration (Neovim 0.11 Native)
vim.api.nvim_create_autocmd('LspAttach', {
  callback = function(args)
    local bufnr = args.buf
    local client = vim.lsp.get_client_by_id(args.data.client_id)

    -- 1. Buffer-local Mappings
    local opts = { buffer = bufnr }
    vim.keymap.set('n', '<leader>e', vim.diagnostic.open_float, { buffer = bufnr, desc = "Line Diagnostics" })

    -- 2. Enable Inlay Hints (0.10+)
    if client:supports_method('textDocument/inlayHint') then
      vim.lsp.inlay_hint.enable(true, { bufnr = bufnr })
    end

    -- 3. Auto-format on Save
    -- Only if the server supports it
    if client:supports_method('textDocument/formatting') then
      vim.api.nvim_create_autocmd('BufWritePre', {
        buffer = bufnr,
        callback = function()
          vim.lsp.buf.format({ bufnr = bufnr, id = client.id })
        end,
      })
    end
  end,
})

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

-- Server Configurations
vim.lsp.config('gopls', {
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

vim.lsp.config('lua_ls', {
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

vim.lsp.config('nixd', {
  settings = {
    nixd = {
      formatting = {
        command = { "nixfmt" }
      },
      nixpkgs = {
        expr = "import <nixpkgs> { }"
      },
      options = {
        home_manager = {
          expr =
          '(builtins.getFlake "${workspaceFolder}").nixosConfigurations.otus.options.home-manager.users.type.getSubOptions [ ]'
        },
        nixos = {
          expr = '(builtins.getFlake "${workspaceFolder}").nixosConfigurations.otus.options'
        }
      }
    }
  },
})

vim.lsp.config('harper_ls', {
  settings = {
    ["harper-ls"] = {
      userDictPath = "~/dict.txt",
    }
  }
})

-- Enable Servers
-- This triggers filetype-based automatic start
vim.lsp.enable({ 'gopls', 'rust_analyzer', 'clangd', 'lua_ls', 'nixd', 'harper_ls' })

-- LSP Floating Window Borders (Rounded)
local border = "rounded"
vim.lsp.handlers["textDocument/hover"] = function(err, result, ctx, config)
  config = config or {}
  config.border = border
  return vim.lsp.handlers.hover(err, result, ctx, config)
end

vim.lsp.handlers["textDocument/signatureHelp"] = function(err, result, ctx, config)
  config = config or {}
  config.border = border
  return vim.lsp.handlers.signature_help(err, result, ctx, config)
end

-- 6. Diagnostics UI
vim.diagnostic.config({
  virtual_text = {
    prefix = '●',
    spacing = 4,
    severity = { min = vim.diagnostic.severity.WARN }, -- Hide HINT (spell checker) and INFO virtual text; keep underlines
  },
  signs = true,
  underline = true,
  update_in_insert = false,
  severity_sort = true,
  float = {
    border = "rounded",
    source = "always",
  },
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

local builtin = require('telescope.builtin')
vim.keymap.set('n', '<leader>ff', builtin.find_files, { desc = "Find Files" })
vim.keymap.set('n', '<leader>fg', builtin.live_grep, { desc = "Live Grep" })

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

-- 8. Autocommands
-- Fast Saving
vim.keymap.set("n", "<leader>w", "<CMD>w<CR>", { desc = "Save File" })

-- Map gotmpl to correct filetype
vim.api.nvim_create_autocmd({ "BufRead", "BufNewFile" }, {
  pattern = "*.gotmpl",
  callback = function()
    vim.bo.filetype = "gotmpl"
  end,
})

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
