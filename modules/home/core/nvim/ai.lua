-- SUPERMAVEN SETUP
require("supermaven-nvim").setup({
  keymaps = {
    accept_suggestion = "<Tab>",
    clear_suggestion = "<C-]>",
    accept_word = "<C-j>",
  },
  ignore_filetypes = { "bigfile", "help" },
  log_level = "info",
})

-- CODECOMPANION SETUP
require("codecompanion").setup({
  strategies = {
    chat = {
      adapter = "gemini",
    },
    inline = {
      adapter = "gemini",
    },
    agent = {
      adapter = "gemini",
    },
  },
  adapters = {
    gemini = function()
      return require("codecompanion.adapters").extend("gemini", {
        env = {
          api_key = "cmd:cat ~/.config/sops-nix/secrets/gemini_api_key",
        },
        schema = {
          model = {
            default = "gemini-3.1-pro-preview",
          },
        },
      })
    end,
  },
  display = {
    chat = {
      window = {
        layout = "vertical",
        width = 0.35,
      },
    },
  },
})

-- KEYBINDINGS
local map = vim.keymap.set

map("n", "<leader>ac", "<cmd>CodeCompanionChat Toggle<cr>", { desc = "AI: Toggle Sidebar Chat" })
map("n", "<leader>ai", "<cmd>CodeCompanion<cr>", { desc = "AI: Inline Assistant" })
map("v", "<leader>ae", ":CodeCompanionChat Add<cr>", { desc = "AI: Add selection to chat" })
map("n", "<leader>aa", "<cmd>CodeCompanionActions<cr>", { desc = "AI: Open Actions Palette" })
