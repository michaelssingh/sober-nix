vim.cmd[[colorscheme tokyonight]]
vim.g.mapleader = " "
vim.opt.number = true
vim.opt.relativenumber = true
vim.opt.expandtab = true
vim.opt.shiftwidth = 2
vim.opt.softtabstop = 2
vim.opt.tabstop = 2
vim.opt.smartindent = true
vim.opt.cursorline = true
vim.opt.ignorecase = true
vim.opt.smartcase = true

-- Lua-specific settings
vim.api.nvim_create_autocmd("FileType", {
  pattern = "lua",
  callback = function()
    vim.opt_local.shiftwidth = 2
    vim.opt_local.tabstop = 2
    vim.opt_local.softtabstop = 2
    vim.opt_local.expandtab = true
  end,
})

require('lspconfig').lua_ls.setup({
  on_init = function(client)
    -- This helps lua_ls understand the Neovim environment
    client.config.settings.Lua = vim.tbl_deep_extend('force', client.config.settings.Lua, {
      runtime = { version = 'LuaJIT' },
      workspace = {
        checkThirdParty = false,
        library = { vim.env.VIMRUNTIME }
      },
    })
  end,
  settings = {
    Lua = {
      diagnostics = {
        globals = { 'vim' }, -- Stop the "undefined global vim" error
      },
      workspace = {
        library = vim.api.nvim_get_runtime_file("", true),
        checkThirdParty = false,
      },
      telemetry = { enable = false },
    },
  },
})

-- Nix Setup
require('lspconfig').nixd.setup({
  settings = {
    nixd = {
      formatting = {
        command = { "nixfmt" },
      },
    },
  },
})

require('nvim-autopairs').setup({})

require("ibl").setup {
  scope = { enabled = true, show_start = true, show_end = false },
  indent = { char = "▏" },
}
require('nvim-treesitter.configs').setup({
  highlight = { enable = true },
  indent = { enable = true }, -- This is the key for "as you type" indentation
})

require('lualine').setup { options = { theme = 'tokyonight' } }

require('gitsigns').setup()

require('which-key').setup()

-- Modern LSP Setup (Fixes 0.11 Deprecation)
require('lspconfig').nixd.setup({
  settings = {
    nixd = {
      formatting = {
        command = { "nixfmt" },
      },
    },
  },
})

-- Configure how diagnostics are displayed
vim.diagnostic.config({
  virtual_text = {
    prefix = '●', -- Small dot at the end of the line
    spacing = 4,
  },
  signs = true,      -- Show icons in the gutter
  underline = true,  -- Underline the actual error
  update_in_insert = false, -- Don't yell at you while you're still typing
  severity_sort = true,
})

-- Change the gutter icons to match Tokyo Night/Nerd Fonts
local signs = { Error = "󰅚 ", Warn = "󰀪 ", Hint = "󰌶 ", Info = "➔ " }
for type, icon in pairs(signs) do
  local hl = "DiagnosticSign" .. type
  vim.fn.sign_define(hl, { text = icon, texthl = hl, numhl = hl })
end

-- Keybind to see the full error in a floating window
vim.keymap.set('n', '<leader>e', vim.diagnostic.open_float, { desc = "Line Diagnostics" })
vim.keymap.set('n', '[d', vim.diagnostic.goto_prev, { desc = "Prev Diagnostic" })
vim.keymap.set('n', ']d', vim.diagnostic.goto_next, { desc = "Next Diagnostic" })

local builtin = require('telescope.builtin')
vim.keymap.set('n', '<leader>ff', builtin.find_files, { desc = "Find Files" })
vim.keymap.set('n', '<leader>fg', builtin.live_grep, { desc = "Live Grep" })
vim.keymap.set('n', 'gd', vim.lsp.buf.definition, { desc = "Go to Definition" })
vim.keymap.set('n', 'K',  vim.lsp.buf.hover, { desc = "Docs" })

-- Format Nixlang on save
vim.api.nvim_create_autocmd("BufWritePre", {
  pattern = { "*.nix", "*.lua" },
  callback = function()
    vim.lsp.buf.format()
  end,
})
