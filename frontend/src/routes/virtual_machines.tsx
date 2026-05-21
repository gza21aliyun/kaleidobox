import type { models } from "../../wailsjs/go/models";
import { useState, useEffect } from "react";
import { useTranslation } from "react-i18next";
import { createRoute } from "@tanstack/react-router";
import { Route as rootRoute } from "./__root";
import { BetterButton } from "../components/ui/BetterButton";
import { BetterSelect } from "../components/ui/BetterSelect";
import { GetAllVMs, AddVM, UpdateVM, DeleteVM, ListVmsInEsx } from "../../wailsjs/go/service/VMService";
import { toast } from "react-hot-toast";
import { VMPanel } from "../components/panel/VMPanel";
import type { appconf } from "../../wailsjs/go/models";
import { useAppStore } from "../store";
import { CollapsibleSection } from "../components/ui/CollapsibleSection";

export const Route = createRoute({
  getParentRoute: () => rootRoute,
  path: "/virtual_machines",
  component: VirtualMachinesPage,
});

function VirtualMachinesPage() {
  const { t } = useTranslation();
  const { config, fetchConfig, updateConfig } = useAppStore();
  const [vms, setVms] = useState<models.Vms[]>([]);
  const [isModalOpen, setIsModalOpen] = useState(false);
  const [editingVm, setEditingVm] = useState<models.Vms | null>(null);
  const [formData, setFormData] = useState<models.Vms>({
    vm_id: "",
    vm_name: "",
    vm_user_name: "",
    vm_pass: "",
    vm_path: "",
    vm_type: "workstation",
    host_url: "",
    host_user: "",
    host_pass: ""
  });
  const [appFormData, setAppFormData] = useState<appconf.AppConfig | null>(null);
  const [esxVms, setEsxVms] = useState<models.Vms[]>([]);
  const [showEsxVmList, setShowEsxVmList] = useState(false);
  const [isLoadingEsxVms, setIsLoadingEsxVms] = useState(false);

  useEffect(() => {
    loadVMs();
  }, []);

  useEffect(() => {
      const init = async () => {
        await fetchConfig();
      };
      init();
    }, [fetchConfig]);

  // 自动保存逻辑
  useEffect(() => {
    if (!appFormData)
      return;

    const hasChanges = JSON.stringify(appFormData) !== JSON.stringify(config);
    if (!hasChanges)
      return;

    const timer = setTimeout(() => {
      updateConfig(appFormData);
    }, 250);

    return () => clearTimeout(timer);
  }, [appFormData, updateConfig, config]);

  useEffect(() => {
    if (config) {
      setAppFormData({ ...config } as appconf.AppConfig);
    }
  }, [config]);
  const loadVMs = async () => {
    try {
      const result = await GetAllVMs();
      setVms(result || []);
    } catch (error) {
      console.error("Failed to load VMs:", error);
      toast.error(t("vm.loadError"));
    }
  };

  const handleAddVM = () => {
    setEditingVm(null);
    setFormData({
      vm_id: "",
      vm_name: "",
      vm_user_name: "",
      vm_pass: "",
      vm_path: "",
      vm_type: "workstation",
      host_url: "",
      host_user: "",
      host_pass: ""
    });
    setIsModalOpen(true);
  };

  const handleEditVM = (vm: models.Vms) => {
    setEditingVm(vm);
    setFormData({ ...vm });
    setIsModalOpen(true);
  };

  const handleDeleteVM = async (vmId: string) => {
    if (window.confirm(t("vm.deleteConfirm"))) {
      try {
        await DeleteVM(vmId);
        toast.success(t("vm.deleteSuccess"));
        loadVMs();
      } catch (error) {
        console.error("Failed to delete VM:", error);
        toast.error(t("vm.deleteError"));
      }
    }
  };

  const handleSaveVM = async () => {
    try {
      if (editingVm) {
        await UpdateVM(formData);
        toast.success(t("vm.updateSuccess"));
      } else {
        await AddVM(formData);
        toast.success(t("vm.addSuccess"));
      }
      setIsModalOpen(false);
      loadVMs();
    } catch (error) {
      console.error("Failed to save VM:", error);
      toast.error(t("vm.saveError"));
    }
  };

  const handleFormChange = (e: React.ChangeEvent<HTMLInputElement | HTMLSelectElement>) => {
    const { name, value } = e.target;
    setFormData({ ...formData, [name]: value });
  };

  const fetchEsxVms = async () => {
    if (formData.vm_type === "esx" && formData.host_url && formData.host_user && formData.host_pass /*&& !formData.vm_path && !formData.vm_name*/) {
      setIsLoadingEsxVms(true);
      try {
        const result = await ListVmsInEsx(formData.host_url, formData.host_user, formData.host_pass);
        setEsxVms(result || []);
        setShowEsxVmList(result && result.length > 0);
      } catch (error) {
        console.error("Failed to fetch ESX VMs:", error);
        toast.error(t("vm.loadError"));
        setEsxVms([]);
        setShowEsxVmList(false);
      } finally {
        setIsLoadingEsxVms(false);
      }
    } else {
      setEsxVms([]);
      setShowEsxVmList(false);
    }
  };

  const handleEsxVmSelect = (vm: models.Vms) => {
    setFormData({
      ...formData,
      vm_name: vm.vm_name,
      vm_path: vm.vm_path || ""
    });
    setShowEsxVmList(false);
  };

  const handleAppFormChange = (newData: appconf.AppConfig) => {
    setAppFormData(newData);
  };

  return (
    <div className="max-w-10xl mx-auto p-8">
      <div className="space-y-6">
        <div className="flex justify-between items-center">
          <h2 className="text-xl font-bold text-brand-700 dark:text-brand-300">{t("vm.title")}</h2>
          <BetterButton onClick={handleAddVM} icon="i-mdi-plus" variant="primary">
            {t("vm.add")}
          </BetterButton>
        </div>

        { appFormData && (
          <CollapsibleSection title="虚拟机设置" icon="i-mdi-server" defaultOpen={false}>
            <VMPanel formData={appFormData} onChange={handleAppFormChange} />
          </CollapsibleSection>
        )}


        <div className="overflow-x-auto">
          <table className="min-w-full divide-y divide-brand-200 dark:divide-brand-700">
            <thead className="bg-brand-50 dark:bg-brand-800">
              <tr>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-brand-500 dark:text-brand-400 uppercase tracking-wider">
                  {t("vm.name")}
                </th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-brand-500 dark:text-brand-400 uppercase tracking-wider">
                  {t("vm.type")}
                </th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-brand-500 dark:text-brand-400 uppercase tracking-wider">
                  {t("vm.path")}
                </th>
                <th scope="col" className="px-6 py-3 text-left text-xs font-medium text-brand-500 dark:text-brand-400 uppercase tracking-wider">
                  {t("vm.host")}
                </th>
                <th scope="col" className="px-6 py-3 text-right text-xs font-medium text-brand-500 dark:text-brand-400 uppercase tracking-wider">
                  {t("common.actions")}
                </th>
              </tr>
            </thead>
            <tbody className="bg-white dark:bg-brand-800 divide-y divide-brand-200 dark:divide-brand-700">
              {vms.map((vm) => (
                <tr key={vm.vm_id}>
                  <td className="px-6 py-4 whitespace-nowrap text-sm font-medium text-brand-900 dark:text-brand-100">
                    {vm.vm_name}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-brand-500 dark:text-brand-400">
                    {vm.vm_type}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-brand-500 dark:text-brand-400">
                    {vm.vm_path || "-"}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-sm text-brand-500 dark:text-brand-400">
                    {vm.host_url || "-"}
                  </td>
                  <td className="px-6 py-4 whitespace-nowrap text-right text-sm font-medium">
                    <div className="flex justify-end gap-2">
                      <BetterButton
                        onClick={() => handleEditVM(vm)}
                        size="sm"
                        icon="i-mdi-pencil"
                      >
                        {t("common.edit")}
                      </BetterButton>
                      <BetterButton
                        onClick={() => handleDeleteVM(vm.vm_id!)}
                        size="sm"
                        variant="danger"
                        icon="i-mdi-trash-can-outline"
                      >
                        {t("common.delete")}
                      </BetterButton>
                    </div>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>

        {isModalOpen && (
          <div className="fixed inset-0 bg-black bg-opacity-50 flex items-center justify-center z-50">
            <div className="bg-white dark:bg-brand-800 rounded-lg p-6 max-w-2xl w-full">
              {/* Header */}
              <div className="flex justify-between items-center mb-4">
                <div className="flex items-center gap-2">
                  <h3 className="text-xl font-semibold text-brand-900 dark:text-white">
                    {editingVm ? t("vm.editTitle") : t("vm.addTitle")}
                  </h3>
                  {formData.vm_type === "esx" && (
                    <BetterButton
                      onClick={async () => {
                        if (!formData.host_url) {
                          toast.error(t("vm.hostUrlPlaceholder"));
                          return;
                        }
                        if (!formData.host_user) {
                          toast.error(t("vm.hostUserPlaceholder"));
                          return;
                        }
                        if (!formData.host_pass) {
                          toast.error(t("vm.hostPasswordPlaceholder"));
                          return;
                        }
                        await fetchEsxVms();
                      }}
                      size="sm"
                      icon="i-mdi-server"
                    >
                      {t("vm.listVms")}
                    </BetterButton>
                  )}
                </div>
                <div className="flex items-center gap-2">
                  
                  <button
                    onClick={() => setIsModalOpen(false)}
                    className="text-gray-500 hover:text-gray-700 dark:text-gray-400 dark:hover:text-gray-200"
                  >
                    ×
                  </button>
                </div>
              </div>

              <div className="space-y-4">
                <div>
                  <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
                    {t("vm.name")}
                  </label>
                  <div className="relative">
                  <input
                    type="text"
                    name="vm_name"
                    value={formData.vm_name}
                    onChange={handleFormChange}
                    onFocus={fetchEsxVms}
                    className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
                    placeholder={t("vm.namePlaceholder")}
                  />
                  {showEsxVmList && (
                    <>
                      <div
                        className="fixed inset-0 z-40"
                        onClick={() => {
                          setShowEsxVmList(false)
                        }}
                      />
                      <div 
                        className="absolute z-50 mt-1 w-full bg-white dark:bg-brand-800 border border-brand-300 dark:border-brand-600 rounded-md shadow-lg max-h-60 overflow-y-auto"
                        onClick={(e) => {
                          e.stopPropagation()
                        }}>
                        {isLoadingEsxVms ? (
                          <div className="px-4 py-2 text-center">{t("vm.loading")}</div>
                        ) : (
                          esxVms.map((vm) => (
                            <div
                              key={vm.vm_name}
                              className="px-4 py-2 hover:bg-brand-100 dark:hover:bg-brand-700 cursor-pointer"
                              onClick={() => handleEsxVmSelect(vm)}
                            >
                              {vm.vm_name}
                            </div>
                          ))
                        )}
                      </div>
                    </>
                  )}
                </div>
                </div>

                <div>
                  <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
                    {t("vm.type")}
                  </label>
                  <BetterSelect
                    value={formData.vm_type ?? ""}
                    onChange={(value) => setFormData({ ...formData, vm_type: value })}
                    options={[
                      { value: "workstation", label: t("vm.typeWorkstation") },
                      { value: "esx", label: t("vm.typeEsx") }
                    ]}
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
                    {t("vm.userName")}
                  </label>
                  <input
                    type="text"
                    name="vm_user_name"
                    value={formData.vm_user_name}
                    onChange={handleFormChange}
                    className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
                    placeholder={t("vm.userNamePlaceholder")}
                  />
                </div>

                <div>
                  <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
                    {t("vm.password")}
                  </label>
                  <input
                    type="password"
                    name="vm_pass"
                    value={formData.vm_pass}
                    onChange={handleFormChange}
                    className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
                    placeholder={t("vm.passwordPlaceholder")}
                  />
                </div>

                {true && (
                  <div>
                    <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
                      {t("vm.path")}
                    </label>
                    <input
                      type="text"
                      name="vm_path"
                      value={formData.vm_path}
                      onChange={handleFormChange}
                      className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
                      placeholder={t("vm.pathPlaceholder")}
                    />
                  </div>
                )}

                {formData.vm_type === "esx" && (
                  <>
                    <div>
                      <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
                        {t("vm.hostUrl")}
                      </label>
                      <input
                        type="text"
                        name="host_url"
                        value={formData.host_url}
                        onChange={handleFormChange}
                        className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
                        placeholder={t("vm.hostUrlPlaceholder")}
                      />
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
                        {t("vm.hostUser")}
                      </label>
                      <input
                        type="text"
                        name="host_user"
                        value={formData.host_user}
                        onChange={handleFormChange}
                        className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
                        placeholder={t("vm.hostUserPlaceholder")}
                      />
                    </div>

                    <div>
                      <label className="block text-sm font-medium text-brand-700 dark:text-brand-300 mb-1">
                        {t("vm.hostPassword")}
                      </label>
                      <input
                        type="password"
                        name="host_pass"
                        value={formData.host_pass}
                        onChange={handleFormChange}
                        className="glass-input w-full px-3 py-2 border border-brand-300 dark:border-brand-600 rounded-md bg-white dark:bg-brand-700 text-brand-900 dark:text-white focus:ring-2 focus:ring-neutral-500 outline-none"
                        placeholder={t("vm.hostPasswordPlaceholder")}
                      />
                    </div>
                  </>
                )}

                <div className="flex justify-end gap-2 pt-4">
                  <BetterButton onClick={() => setIsModalOpen(false)}>
                    {t("common.cancel")}
                  </BetterButton>
                  <BetterButton onClick={handleSaveVM} variant="primary">
                    {t("common.save")}
                  </BetterButton>
                </div>
              </div>
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
